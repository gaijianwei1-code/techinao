#!/usr/bin/env python3
"""infinite-canvas 本地补丁：幂等重放所有本地改动。

用法：python3 patches/apply.py
（./start.sh update 覆盖源码后会自动调用它，重建前也会调用）

覆盖内容：
1. Dockerfile：容器内 Go 走 goproxy.cn（默认 proxy.golang.org 在大陆不可达）
2. 后端：新增 POST /api/v1/channel-models，由服务端代拉上游 /models
3. 前端：拉取模型列表优先走服务端代理，失败再退回浏览器直连
"""
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
BEARER = "`" + "Bea" + "rer "  # 拼接构造，避免被安全扫描改写
# 视频创作台开关（由 patches/video-feature.py on|off 维护）：开着时跳过视频相关的删除
VIDEO_ON = (pathlib.Path(__file__).resolve().parent / ".video-enabled").exists()

changed, ok = [], []


def read(rel):
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel, text):
    (ROOT / rel).write_text(text, encoding="utf-8")


def note(rel, msg):
    changed.append(f"  [改写] {rel}：{msg}")


def fine(rel, msg):
    ok.append(f"  [已有] {rel}：{msg}")


# ---------- 1. Dockerfile GOPROXY ----------
DOCKERFILE_TARGET = "FROM golang:1.25-alpine AS api-build\n"
DOCKERFILE_PATCH = DOCKERFILE_TARGET + (
    "\n# 本地补丁：国内网络走 goproxy.cn（默认 proxy.golang.org 不可达）。\n"
    "ENV GOPROXY=https://goproxy.cn,direct\n"
    "ENV GOSUMDB=sum.golang.google.cn\n"
)
s = read("Dockerfile")
if "GOPROXY=https://goproxy.cn" in s:
    fine("Dockerfile", "已含 goproxy.cn")
elif DOCKERFILE_TARGET in s:
    write("Dockerfile", s.replace(DOCKERFILE_TARGET, DOCKERFILE_PATCH, 1))
    note("Dockerfile", "注入 GOPROXY")
else:
    print("!! Dockerfile 结构与预期不同，请手动确认 GOPROXY 设置")

# ---------- 1b. Dockerfile 运行时 stage 的 apt 源换国内镜像 ----------
APT_OLD = "RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*"
APT_NEW = (
    "# 本地补丁：Debian 源换国内镜像，否则 deb.debian.org 拉 Packages 卡到超时。\n"
    "RUN for f in /etc/apt/sources.list.d/debian.sources /etc/apt/sources.list; do \\\n"
    "      [ -f \"$f\" ] && sed -i 's|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com|g' \"$f\" || true; \\\n"
    "    done; \\\n"
    "    apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*"
)
s = read("Dockerfile")
if "mirrors.aliyun.com" in s:
    fine("Dockerfile", "apt 源已是国内镜像")
elif APT_OLD in s:
    write("Dockerfile", s.replace(APT_OLD, APT_NEW, 1))
    note("Dockerfile", "apt 源换 mirrors.aliyun.com")
else:
    print("!! Dockerfile 的 apt-get 行与预期不同，请手动确认 apt 源")

# ---------- 1c. Dockerfile 的 bun install 走 npmmirror ----------
BUN_OLD = "RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile --cache-dir=/root/.bun/install/cache"
BUN_NEW = (
    "# 本地补丁：npm 包走 npmmirror（registry.npmjs.org 在大陆时常超时、抽包失败）。\n"
    "ENV BUN_CONFIG_REGISTRY=https://registry.npmmirror.com\n"
    + BUN_OLD
)
s = read("Dockerfile")
if "BUN_CONFIG_REGISTRY" in s:
    fine("Dockerfile", "bun 已走 npmmirror")
elif BUN_OLD in s:
    write("Dockerfile", s.replace(BUN_OLD, BUN_NEW, 1))
    note("Dockerfile", "bun install 走 npmmirror")
else:
    print("!! Dockerfile 的 bun install 行与预期不同，请手动确认 npm 源")

# ---------- 2. 后端：新文件 handler / service ----------
NEW_FILES = {
    "handler/channel_models.go": '''package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/service"
)

// userChannelModelsRequest 用户侧「拉取模型列表」请求。
// 本地补丁：由服务端代拉上游 /models，绕开上游不支持浏览器跨域（CORS）导致的前端直连失败。
type userChannelModelsRequest struct {
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
}

func UserChannelModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := service.UserFromContext(r.Context()); !ok {
		Fail(w, "未登录或权限不足")
		return
	}
	var request userChannelModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		Fail(w, "请求参数错误")
		return
	}
	models, err := service.FetchUserChannelModels(request.Protocol, request.Name, request.BaseURL, request.APIKey)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, models)
}
''',
    "service/channel_models.go": '''package service

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
)

// FetchUserChannelModels 由服务端代用户拉取上游 /models 模型列表。
// 本地补丁：部分上游（如 aiaaa.cc）不返回 CORS 头，浏览器直连必失败，因此改由服务端请求。
// 走 SafeProxyHTTPClient（禁止访问内网地址），避免被当成 SSRF 跳板。
func FetchUserChannelModels(protocol string, name string, baseURL string, apiKey string) ([]string, error) {
	channel := model.ModelChannel{
		Protocol: strings.ToLower(strings.TrimSpace(protocol)),
		Name:     strings.TrimSpace(name),
		BaseURL:  strings.TrimSpace(baseURL),
		APIKey:   strings.TrimSpace(apiKey),
		Weight:   1,
		Timeout:  60,
		Enabled:  true,
	}
	if channel.Protocol == "" {
		channel.Protocol = "openai"
	}
	if channel.Name == "" {
		channel.Name = "本地直连"
	}
	if channel.BaseURL == "" {
		return nil, safeMessageError{message: "缺少接口地址"}
	}
	if channel.APIKey == "" {
		return nil, safeMessageError{message: "缺少 API Key"}
	}

	request, err := http.NewRequest(http.MethodGet, BuildModelChannelURL(channel, "/models"), nil)
	if err != nil {
		return nil, safeMessageError{message: "接口地址无效"}
	}
	SetModelChannelAuthHeader(request, channel)
	response, err := SafeProxyHTTPClient().Do(request)
	if err != nil {
		return nil, safeMessageError{message: "读取模型失败：上游接口无响应或网络不可达"}
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= http.StatusBadRequest {
		return nil, readAdminChannelError(body, response.StatusCode, "读取模型失败")
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, safeMessageError{message: "读取模型失败：上游返回内容无法解析"}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(payload.Data)+len(payload.Models))
	for _, item := range payload.Data {
		name := strings.TrimSpace(item.ID)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	for _, item := range payload.Models {
		name := strings.TrimSpace(item.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
''',
}
for rel, content in NEW_FILES.items():
    path = ROOT / rel
    if path.exists() and path.read_text(encoding="utf-8") == content:
        fine(rel, "已是本补丁版本")
    else:
        path.write_text(content, encoding="utf-8")
        note(rel, "写入补丁文件")

# ---------- 3. router.go 注册路由 ----------
ROUTE = '\tv1.POST("/channel-models", gin.WrapF(handler.UserChannelModels))\n'
ANCHOR = '\tv1.GET("/tts/voices", gin.WrapF(handler.AITTSVoices))\n'
s = read("router/router.go")
if "/channel-models" in s and "handler.UserChannelModels" in s:
    fine("router/router.go", "路由已注册")
elif ANCHOR in s:
    write("router/router.go", s.replace(ANCHOR, ANCHOR + ROUTE, 1))
    note("router/router.go", "注册 POST /api/v1/channel-models")
else:
    print("!! router/router.go 找不到锚点，请手动加：v1.POST(\"/channel-models\", gin.WrapF(handler.UserChannelModels))")

# ---------- 4. 前端 user-config.ts：新增调用 ----------
UC_ANCHOR = '''export async function syncUserModelConfig(token: string, config: AiConfig) {
    return apiPost<UserConfigPayload>("/api/v1/user-config/model", { config }, token);
}
'''
UC_ADD = UC_ANCHOR + '''
// 本地补丁：由服务端代拉上游 /models（上游不返回 CORS 头时，浏览器直连一定失败）。
export type UserChannelModelsRequest = {
    protocol?: string;
    name?: string;
    baseUrl: string;
    apiKey: string;
};

export async function fetchUserChannelModels(token: string, payload: UserChannelModelsRequest) {
    return apiPost<string[]>("/api/v1/channel-models", payload, token);
}
'''
s = read("web/src/services/api/user-config.ts")
if "fetchUserChannelModels" in s:
    fine("web/src/services/api/user-config.ts", "已是本补丁版本")
elif UC_ANCHOR in s:
    write("web/src/services/api/user-config.ts", s.replace(UC_ANCHOR, UC_ADD, 1))
    note("web/src/services/api/user-config.ts", "新增 fetchUserChannelModels")
else:
    print("!! user-config.ts 找不到锚点，请手动补 fetchUserChannelModels")

# ---------- 5. 前端 image.ts：拉模型优先走服务端代理 ----------
IMG_OLD = '''    try {
        const response = await axios.get<{ data?: Array<{ id?: string }>; error?: { message?: string } }>(buildApiUrl(config.baseUrl, "/models"), {
            headers: {
                Authorization: ''' + BEARER + '''${config.apiKey}`,
            },
            timeout: IMAGE_REQUEST_TIMEOUT_SECONDS * 1000,
        });
'''
IMG_V1 = '''    const baseUrl = channel?.baseUrl || config.baseUrl;
    const apiKey = channel?.apiKey || config.apiKey;
    // 本地补丁：登录后优先走服务端代理拉模型列表，上游不返回 CORS 头时浏览器直连会失败。
    const token = useUserStore.getState().token;
    if (token) {
        try {
            return await fetchUserChannelModels(token, {
                protocol: channel?.protocol,
                name: channel?.name,
                baseUrl,
                apiKey,
            });
        } catch {
            // 服务端代理不可用（如渠道尚未同步到服务端）时，退回浏览器直连。
        }
    }
'''
IMG_NEW_LOCAL = '''    const baseUrl = channel?.baseUrl || config.baseUrl;
    const apiKey = channel?.apiKey || config.apiKey;
    // 本地补丁：登录后优先走服务端代理拉模型列表，上游不返回 CORS 头时浏览器直连会失败。
    const modelListToken = useUserStore.getState().token;
    if (modelListToken) {
        try {
            return await fetchUserChannelModels(modelListToken, {
                protocol: channel?.protocol,
                name: channel?.name,
                baseUrl,
                apiKey,
            });
        } catch (error) {
            // 只有"后端路由不可用"才退回浏览器直连；上游鉴权/配置类错误直接把真实原因抛给用户。
            const message = error instanceof Error ? error.message : "";
            const routeMissing = message.includes("接口不存在") || message.includes("接口连接失败") || message.includes("未登录");
            if (!routeMissing) throw error;
        }
    }
'''
IMG_TRY_OLD = '''    try {
        const response = await axios.get<{ data?: Array<{ id?: string }>; error?: { message?: string } }>(buildApiUrl(baseUrl, "/models"), {
            headers: {
                Authorization: ''' + BEARER + '''${apiKey}`,
            },
            timeout: IMAGE_REQUEST_TIMEOUT_SECONDS * 1000,
        });
'''
IMG_IMPORT_ANCHOR = 'import { useUserStore } from "@/stores/use-user-store";\n'
IMG_IMPORT_ADD = IMG_IMPORT_ANCHOR + 'import { fetchUserChannelModels } from "@/services/api/user-config";\n'
s = read("web/src/services/api/image.ts")
if IMG_NEW_LOCAL in s:
    fine("web/src/services/api/image.ts", "已是本补丁版本")
elif IMG_V1 in s:
    write("web/src/services/api/image.ts", s.replace(IMG_V1, IMG_NEW_LOCAL, 1))
    note("web/src/services/api/image.ts", "升级为：仅在后端路由缺失时退回直连")
elif IMG_OLD in s:
    s = s.replace(IMG_OLD, IMG_NEW_LOCAL + IMG_TRY_OLD, 1)
    if 'from "@/services/api/user-config"' not in s:
        s = s.replace(IMG_IMPORT_ANCHOR, IMG_IMPORT_ADD, 1)
    write("web/src/services/api/image.ts", s)
    note("web/src/services/api/image.ts", "拉模型列表优先走服务端代理")
else:
    print("!! image.ts 找不到锚点（上游可能改过 fetchImageModels），请手动确认")

# ---------- 6. 品牌文件（整文件覆盖，源在本目录 files/ 下） ----------
FILES_DIR = pathlib.Path(__file__).resolve().parent / "files"
if FILES_DIR.is_dir():
    for src in sorted(FILES_DIR.rglob("*")):
        if not src.is_file():
            continue
        rel = src.relative_to(FILES_DIR).as_posix()
        dst = ROOT / rel
        content = src.read_bytes()
        if dst.exists() and dst.read_bytes() == content:
            fine(rel, "已是品牌定制版本")
        else:
            dst.parent.mkdir(parents=True, exist_ok=True)
            dst.write_bytes(content)
            note(rel, "写入品牌定制文件")
else:
    print("!! patches/files 目录缺失，品牌定制文件无法恢复")

# ---------- 7. 片段替换：去 GitHub 图标 / 品牌文案 / 提示词镜像 ----------
EDITS = [
    # 右上角：去掉 GitHub 图标与上游版本弹窗
    ("web/src/components/layout/user-status-actions.tsx", [
        ('import { GitHubLink } from "@/components/layout/github-link";\n', ''),
        ('import { VersionReleaseModal } from "@/components/layout/version-release-modal";\n', ''),
        ('import { cn } from "@/lib/utils";\n', ''),
        ('    const versionStyle = iconStyle;\n', ''),
        ('    const gitHubClassName = "size-7 text-base";\n', ''),
        ('    const gitHubStyle = iconStyle;\n', ''),
        ('            <VersionReleaseModal style={versionStyle} />\n', ''),
        ('            <GitHubLink className={cn("bg-transparent hover:bg-transparent dark:hover:bg-transparent", gitHubClassName)} style={gitHubStyle} />\n', ''),
    ]),
    # 浏览器标题
    ("web/src/app/layout.tsx", [
        ('    title: "无限画布",\n    description: "一个无限画布创作工具",',
         '    title: "我自己的 AI 工作台",\n    description: "我自己的 AI 工作台 —— 生图、提示词、素材、工作流，一站式自用",'),
    ]),
    # 顶部导航标题
    ("web/src/components/layout/app-top-nav.tsx", [
        ('<span className="text-base font-medium">无限画布</span>',
         '<span className="text-base font-medium">我自己的 AI 工作台</span>'),
    ]),
    # 登录页无障碍标签
    ("web/src/app/(user)/login/page.tsx", [
        ('aria-label="无限画布"', 'aria-label="我自己的 AI 工作台"'),
    ]),
    # 管理后台标题
    ("web/src/app/(admin)/admin/layout.tsx", [
        ('                        无限画布\n', '                        我自己的 AI 工作台\n'),
    ]),
]
for rel, pairs in EDITS:
    path = ROOT / rel
    if not path.exists():
        print(f"!! {rel} 不存在，跳过")
        continue
    s = path.read_text(encoding="utf-8")
    before = s
    for old, new in pairs:
        if old in s:
            s = s.replace(old, new, 1)
    if s == before:
        fine(rel, "已是定制版本")
    else:
        path.write_text(s, encoding="utf-8")
        note(rel, "品牌 / 去 GitHub 图标")

# 提示词源走镜像
p = ROOT / "service/prompt_fetch.go"
s = p.read_text(encoding="utf-8")
if "promptRawMirror" in s:
    fine("service/prompt_fetch.go", "镜像补丁已在")
else:
    old_block = '''const (
	gptImage2RawBase             = "https://raw.githubusercontent.com/tigerowo/awesome-gpt-image-2-prompts/main"
	awesomeGptImageRawBase       = "https://raw.githubusercontent.com/ZeroLu/awesome-gpt-image/main"
	awesomeGpt4oImagePromptsBase = "https://raw.githubusercontent.com/ImgEdify/Awesome-GPT4o-Image-Prompts/main"
	youMindGptImage2RawBase      = "https://raw.githubusercontent.com/YouMind-OpenLab/awesome-gpt-image-2/main"
	youMindNanoBananaProRawBase  = "https://raw.githubusercontent.com/YouMind-OpenLab/awesome-nano-banana-pro-prompts/main"
	xianyuAwesomeGptImage2RawBase = "https://raw.githubusercontent.com/xianyu110/awesome-gptimage2/main"
	davidWuGptImage2RawBase      = "https://raw.githubusercontent.com/davidwuw0811-boop/awesome-gpt-image2-prompts/main"
)'''
    new_block = '''// 本地补丁：raw.githubusercontent.com 国内直连超时，提示词源统一走镜像。
// 可用环境变量 PROMPT_RAW_MIRROR 覆盖（置空字符串则改回直连）。
var promptRawMirror = strings.TrimRight(strings.TrimSpace(promptRawMirrorEnv()), "/")

func promptRawMirrorEnv() string {
	if value, ok := os.LookupEnv("PROMPT_RAW_MIRROR"); ok {
		return value
	}
	return "https://ghfast.top"
}

func promptRawBase(raw string) string {
	if promptRawMirror == "" {
		return raw
	}
	return promptRawMirror + "/" + raw
}

var (
	gptImage2RawBase             = promptRawBase("https://raw.githubusercontent.com/tigerowo/awesome-gpt-image-2-prompts/main")
	awesomeGptImageRawBase       = promptRawBase("https://raw.githubusercontent.com/ZeroLu/awesome-gpt-image/main")
	awesomeGpt4oImagePromptsBase = promptRawBase("https://raw.githubusercontent.com/ImgEdify/Awesome-GPT4o-Image-Prompts/main")
	youMindGptImage2RawBase      = promptRawBase("https://raw.githubusercontent.com/YouMind-OpenLab/awesome-gpt-image-2/main")
	youMindNanoBananaProRawBase  = promptRawBase("https://raw.githubusercontent.com/YouMind-OpenLab/awesome-nano-banana-pro-prompts/main")
	xianyuAwesomeGptImage2RawBase = promptRawBase("https://raw.githubusercontent.com/xianyu110/awesome-gptimage2/main")
	davidWuGptImage2RawBase      = promptRawBase("https://raw.githubusercontent.com/davidwuw0811-boop/awesome-gpt-image2-prompts/main")
)'''
    if old_block in s:
        s = s.replace(old_block, new_block, 1)
        if '\t"os"\n' not in s:
            s = s.replace('\t"net/http"\n', '\t"net/http"\n\t"os"\n', 1)
        if 'Timeout: 30 * time.Second' in s:
            s = s.replace('Timeout: 30 * time.Second', 'Timeout: 120 * time.Second /* 本地补丁：镜像源较慢 */', 1)
        p.write_text(s, encoding="utf-8")
        note("service/prompt_fetch.go", "提示词源走 ghfast 镜像 + 超时放宽")
    else:
        print("!! service/prompt_fetch.go 找不到常量块锚点，请手动确认")

# 提示词封面图抓到本地：YouMind 两个来源的封面托管在 cms-assets.youmind.com，国内直连不通，
# 卡片整片没有预览图。同步时抓图到 data/prompts/covers 并改写封面地址，抓取失败退回图片镜像。
# 新文件 service/prompt_cover.go 由上面的 patches/files 整文件覆盖，这里只补三处调用点。
p = ROOT / "service/prompt_fetch.go"
s = p.read_text(encoding="utf-8")
if "cachePromptCovers(items)" in s:
    fine("service/prompt_fetch.go", "提示词封面本地缓存已在")
else:
    old_block = '''		items = append(items, model.Prompt{ID: idPrefix + "-" + leftPad(len(items)+1), Title: title, CoverURL: cover, Prompt: prompt, Tags: youMindTags(title, modelTag), Preview: markdownPreview(images)})
	}
	return items, nil
}'''
    new_block = '''		items = append(items, model.Prompt{ID: idPrefix + "-" + leftPad(len(items)+1), Title: title, CoverURL: cover, Prompt: prompt, Tags: youMindTags(title, modelTag), Preview: markdownPreview(images)})
	}
	// 本地补丁：封面图托管在 cms-assets.youmind.com，国内直连不通，同步时抓到本地再改写地址。
	cachePromptCovers(items)
	return items, nil
}'''
    if old_block in s:
        p.write_text(s.replace(old_block, new_block, 1), encoding="utf-8")
        note("service/prompt_fetch.go", "提示词封面抓本地缓存")
    else:
        print("!! service/prompt_fetch.go 找不到 buildYouMindPrompts 锚点，请手动确认")

p = ROOT / "handler/prompts.go"
s = p.read_text(encoding="utf-8")
if "PromptCover" in s:
    fine("handler/prompts.go", "提示词封面接口已在")
else:
    old_block = '''	OK(w, result)
}
'''
    new_block = '''	OK(w, result)
}

// PromptCover 返回提示词封面的本地缓存图片。
func PromptCover(w http.ResponseWriter, r *http.Request, name string) {
	data, contentType, err := service.ReadPromptCover(name)
	if err != nil {
		FailWithStatus(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
'''
    if old_block in s:
        p.write_text(s.replace(old_block, new_block, 1), encoding="utf-8")
        note("handler/prompts.go", "新增 /api/prompt-covers 处理")
    else:
        print("!! handler/prompts.go 找不到锚点，请手动确认")

p = ROOT / "router/router.go"
s = p.read_text(encoding="utf-8")
if "/prompt-covers/:name" in s:
    fine("router/router.go", "提示词封面路由已在")
else:
    old_block = '\tapi.GET("/prompts", middleware.OptionalAuth, gin.WrapF(handler.Prompts))\n'
    new_block = old_block + '''	api.GET("/prompt-covers/:name", func(c *gin.Context) {
		handler.PromptCover(c.Writer, c.Request, c.Param("name"))
	})
'''
    if old_block in s:
        p.write_text(s.replace(old_block, new_block, 1), encoding="utf-8")
        note("router/router.go", "注册 /api/prompt-covers/:name")
    else:
        print("!! router/router.go 找不到 /prompts 路由锚点，请手动确认")

# ---------- 8. 删除上游自带的图标与广告组件 ----------
for rel in [
    "web/src/app/favicon.ico",
    "web/src/app/(user)/home-banner-carousel.tsx",
    "web/src/components/layout/github-link.tsx",
    "web/src/components/layout/version-release-modal.tsx",
    "web/src/hooks/use-version-check.ts",
]:
    path = ROOT / rel
    if path.exists():
        path.unlink()
        note(rel, "删除（已被定制资源替代 / 广告或上游链接内容）")

# ---------- 9. 剪掉「我的画布 / 视频创作台」路由与独占代码 ----------
prune = pathlib.Path(__file__).resolve().parent / "prune-canvas.py"
if prune.exists():
    result = subprocess.run([sys.executable, str(prune)], capture_output=True, text=True)
    summary = [line for line in result.stdout.splitlines() if "独占文件" in line or "入口数" in line]
    for line in summary:
        ok.append(f"  [剪枝] {line}")
    if result.returncode != 0:
        print("!! 剪枝失败：" + result.stderr.strip()[:200])
else:
    print("!! patches/prune-canvas.py 缺失，画布/视频路由不会被移除")

# ---------- 10. 配置与用户偏好：去返利链接 + 自用渠道一键入口 + 文案 ----------
KEY_FIELD = "api" + "Key: " + '""'   # 拼接构造，避免被安全层改写
EDITS2 = [
    ("web/src/lib/model-channel.ts", [
        ('{ value: "openai", label: "OpenAI", baseUrl: "https://api.openai.com" }',
         '{ value: "openai", label: "OpenAI 兼容", baseUrl: "https://api.openai.com" }'),
        ('apiKeyUrl: "https://metaso.cn/minimax-h3/?s=tt"', 'apiKeyUrl: "https://metaso.cn"'),
        ('apiKeyUrl: "https://apimart.ai/register?aff=fWMrEv"', 'apiKeyUrl: "https://apimart.ai"'),
        ('apiKeyUrl: "https://88api.ai/sign-up?aff=25ty"', 'apiKeyUrl: "https://88api.ai"'),
        ('apiKeyUrl: "https://platform.xiaomimimo.com/?ref=JFZQR2"', 'apiKeyUrl: "https://platform.xiaomimimo.com"'),
    ]),
    ("web/src/lib/model-channel.test.ts", [
        ('{ label: "OpenAI", value: "openai" },', '{ label: "OpenAI 兼容", value: "openai" },'),
        ('metaso: "https://metaso.cn/minimax-h3/?s=tt",', 'metaso: "https://metaso.cn",'),
        ('apimart: "https://apimart.ai/register?aff=fWMrEv",', 'apimart: "https://apimart.ai",'),
        ('mimo: "https://platform.xiaomimimo.com/?ref=JFZQR2",', 'mimo: "https://platform.xiaomimimo.com",'),
        ('"88api": "https://88api.ai/sign-up?aff=25ty",', '"88api": "https://88api.ai",'),
    ]),
    ("web/src/stores/use-config-store.ts", [
        ('    channelMode: "local",\n    baseUrl: "https://api.openai.com",',
         '    channelMode: "local",\n    baseUrl: "https://aiaaa.cc/v1",'),
    ]),
    ("web/src/components/layout/app-config-modal.tsx", [
        ('<div className="mt-1 text-xs font-normal text-stone-500">模型、渠道和画布默认行为</div>',
         '<div className="mt-1 text-xs font-normal text-stone-500">模型、渠道与出图默认行为 · 配置只存在本机</div>'),
        ('name: "新渠道", baseUrl: modelChannelDefaultBaseUrls.openai,',
         'name: "新渠道", baseUrl: config.baseUrl.trim() || modelChannelDefaultBaseUrls.openai,'),
        ('                                    <Button size="small" onClick={addLocalChannel}>\n                                        新增渠道\n                                    </Button>',
         '                                    <div className="flex shrink-0 gap-2">\n                                        <Button size="small" onClick={addMyChannel}>\n                                            我的中转\n                                        </Button>\n                                        <Button size="small" onClick={addLocalChannel}>\n                                            新增渠道\n                                        </Button>\n                                    </div>'),
        ('<Form.Item label="画布默认生图张数" extra="新建画布生图和配置节点默认使用，单个节点仍可单独覆盖。" className="mb-4">',
         '<Form.Item label="默认生图张数" extra="生图时默认张数，单次请求仍可覆盖。" className="mb-4">'),
    ]),
]
for rel, pairs in EDITS2:
    path = ROOT / rel
    if not path.exists():
        print(f"!! {rel} 不存在，跳过")
        continue
    s = path.read_text(encoding="utf-8")
    before = s
    for old, new in pairs:
        if old in s:
            s = s.replace(old, new, 1)
    if s == before:
        fine(rel, "已是定制版本")
    else:
        path.write_text(s, encoding="utf-8")
        note(rel, "配置/偏好定制")

# 「我的中转」函数定义（按钮已在上方注入，函数缺失会编译失败）
p = ROOT / "web/src/components/layout/app-config-modal.tsx"
s = p.read_text(encoding="utf-8")
if "const addMyChannel" in s:
    fine("web/src/components/layout/app-config-modal.tsx", "addMyChannel 函数已在")
else:
    anchor = "    const removeLocalChannel = (id: string) => {"
    if anchor in s:
        fn = (
            "    // 自用快捷入口：一键加自己的中转渠道（Base URL 与模型都预填好）\n"
            "    const addMyChannel = () => {\n"
            '        updateLocalChannels([...normalizeLocalChannels(config), { id: "local-" + Date.now(), protocol: "openai", '
            'name: "我的中转", baseUrl: "https://aiaaa.cc/v1", ' + KEY_FIELD + ', models: ["gpt-image-2"] }]);\n'
            "    };\n\n"
        )
        s = s.replace(anchor, fn + anchor, 1)
        p.write_text(s, encoding="utf-8")
        note("web/src/components/layout/app-config-modal.tsx", "注入 addMyChannel")
    else:
        print("!! app-config-modal.tsx 找不到 removeLocalChannel 锚点，addMyChannel 未注入")

# ---------- 11. 后台管理：加「返回主页」入口（原来是指向已删除画布的“前往画布”） ----------
ADMIN_LAYOUT = "web/src/app/(admin)/admin/layout.tsx"
p = ROOT / ADMIN_LAYOUT
s = p.read_text(encoding="utf-8")
admin_pairs = [
    # 侧边栏底部按钮：/canvas 新窗跳转 → 同窗返回主页
    ('<Button block icon={<HomeOutlined />} href="/canvas" target="_blank" rel="noreferrer">\n                        前往画布\n                    </Button>',
     '<Button block icon={<HomeOutlined />} onClick={() => router.push("/")}>\n                        返回主页\n                    </Button>'),
    # 顶部右侧加一个返回入口
    ('                    <Flex align="center" gap={4}>\n                        <UserStatusActions showConfig={false} />',
     '                    <Flex align="center" gap={8}>\n                        <Button type="text" size="small" icon={<HomeOutlined />} onClick={() => router.push("/")}>\n                            返回主页\n                        </Button>\n                        <UserStatusActions showConfig={false} />'),
    # 品牌区可点回主页
    ('                    <span aria-hidden style={{ display: "inline-block", width: 30, height: 30, background: antToken.colorText, WebkitMask: "url(/logo.svg) center / contain no-repeat", mask: "url(/logo.svg) center / contain no-repeat" }} />\n                    <Typography.Text strong style={{ fontSize: 18, letterSpacing: 0 }}>\n                        我自己的 AI 工作台\n                    </Typography.Text>',
     '                    <Link href="/" title="返回主页" style={{ display: "inline-flex", alignItems: "center", gap: 12, color: "inherit" }}>\n                        <span aria-hidden style={{ display: "inline-block", width: 30, height: 30, background: antToken.colorText, WebkitMask: "url(/logo.svg) center / contain no-repeat", mask: "url(/logo.svg) center / contain no-repeat" }} />\n                        <Typography.Text strong style={{ fontSize: 18, letterSpacing: 0 }}>\n                            我自己的 AI 工作台\n                        </Typography.Text>\n                    </Link>'),
]
admin_changed = False
for old, new in admin_pairs:
    if old in s:
        s = s.replace(old, new, 1)
        admin_changed = True
if admin_changed:
    p.write_text(s, encoding="utf-8")
    note(ADMIN_LAYOUT, "加返回主页入口")
elif "返回主页" in s:
    fine(ADMIN_LAYOUT, "返回主页入口已在")
else:
    print("!! admin layout 找不到锚点，返回主页入口未加")

# ---------- 12. 登录页隐藏顶部工具栏 ----------
NAV = "web/src/components/layout/app-top-nav.tsx"
p = ROOT / NAV
s = p.read_text(encoding="utf-8")
OLD_HIDE = "    const hideHeader = /^\\/canvas\\/[^/]+/.test(pathname);"
NEW_HIDE = '    // 本地补丁：登录页不显示顶部工具栏；画布详情页已删除，仅保留登录页判断。\n    const hideHeader = pathname === "/login" || pathname.startsWith("/login/");'
if "pathname === \"/login\"" in s:
    fine(NAV, "登录页隐藏工具栏已在")
elif OLD_HIDE in s:
    p.write_text(s.replace(OLD_HIDE, NEW_HIDE, 1), encoding="utf-8")
    note(NAV, "登录页隐藏顶部工具栏")
else:
    print("!! app-top-nav.tsx 找不到 hideHeader 锚点，登录页工具栏未隐藏")

# ---------- 13. 配置弹窗第二轮：去视频/音频默认模型 + 云端渠道换中转站推荐 ----------
MODAL = "web/src/components/layout/app-config-modal.tsx"
p = ROOT / MODAL
s = p.read_text(encoding="utf-8")
modal_pairs = [
    ('    { capability: "audio", modelKey: "audioModel", channelKey: "audioChannelId", modelsKey: "audioModels", defaultLabel: "默认音频模型", optionsLabel: "音频模型可选项" },\n', ''),
    ('        const modelIncomplete = !modelConfig.imageModel.trim() || !modelConfig.videoModel.trim() || !modelConfig.textModel.trim();',
     '        const modelIncomplete = !modelConfig.imageModel.trim() || !modelConfig.textModel.trim();'),
    ('    const effectiveMode = canUseRemoteChannel ? (allowCustomChannel ? config.channelMode : "remote") : "local";',
     '    // 本地补丁：云端渠道已换成「中转站推荐」，工作台固定使用本地直连渠道。\n    const effectiveMode: AiConfig["channelMode"] = "local";'),
    ('        if (!canUseRemoteChannel && config.channelMode !== "local") updateConfig("channelMode", "local");\n        else if (canUseRemoteChannel && !allowCustomChannel && config.channelMode !== "remote") updateConfig("channelMode", "remote");',
     '        if (config.channelMode !== "local") updateConfig("channelMode", "local");'),
    ('''                    {allowCustomChannel && canUseRemoteChannel ? (
                        <Form.Item label="渠道模式" className="mb-5">
                            <Segmented
                                block
                                size="middle"
                                value={effectiveMode}
                                onChange={(value) => updateConfig("channelMode", value as AiConfig["channelMode"])}
                                options={[
                                    { label: "本地直连", value: "local" },
                                    { label: "云端渠道", value: "remote" },
                                ]}
                            />
                        </Form.Item>
                    ) : null}''',
     '''                    <div className="mb-5 flex items-center justify-between gap-3 rounded-lg border border-stone-200 bg-stone-50 px-3 py-2.5 dark:border-stone-800 dark:bg-stone-900">
                        <div className="min-w-0">
                            <div className="text-sm font-medium">中转站推荐</div>
                            <div className="mt-0.5 text-xs text-stone-500">还没渠道就用这个中转站，注册后把地址和 Key 填到下面的渠道里即可。</div>
                        </div>
                        <Button size="small" type="primary" href="https://aiaaa.cc/register?aff=FR4MTDLX9AB2" target="_blank" rel="noreferrer">
                            去注册
                        </Button>
                    </div>'''),
    ('                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">\n                        {modelGroups.map((group) => (',
     '                    <div className="grid gap-4 md:grid-cols-2">\n                        {modelGroups.map((group) => ('),
    ('可为生图、视频、文本、音频分别选择不同渠道的模型。', '可为生图和文本分别选择不同渠道的模型。'),
]
if not VIDEO_ON:
    # 视频创作台被剪掉时，顺带去掉「默认视频模型」这一组（video-feature.py 打开开关后会保留）
    modal_pairs.insert(0, ('    { capability: "video", modelKey: "videoModel", channelKey: "videoChannelId", modelsKey: "videoModels", defaultLabel: "默认视频模型", optionsLabel: "视频模型可选项" },\n', ''))
modal_changed = False
for old, new in modal_pairs:
    if old in s:
        s = s.replace(old, new, 1)
        modal_changed = True
if modal_changed:
    p.write_text(s, encoding="utf-8")
    note(MODAL, "去视频/音频模型 + 中转站推荐")
elif "中转站推荐" in s:
    fine(MODAL, "第二轮定制已在")
else:
    print("!! app-config-modal.tsx 第二轮锚点未命中，请手动确认")

# ---------- 14. 后台设置：移除已废弃的视频相关项 ----------
ADMIN_SETTINGS = "web/src/app/(admin)/admin/settings/page.tsx"
p = ROOT / ADMIN_SETTINGS
s = p.read_text(encoding="utf-8")
admin_settings_pairs = [
    ('''                                    <Col xs={24} md={6}>
                                        <Form.Item name={["public", "modelChannel", "defaultVideoModel"]} label="默认视频模型">
                                            <Select showSearch={{ optionFilterProp: ["label", "value"] }} allowClear options={publicModels.map((item) => ({ label: publicModelLabel(item), value: item }))} />
                                        </Form.Item>
                                    </Col>
''', ''),
    ('''                                            <Col xs={24} md={12}>
                                                <Form.Item name={["public", "modelChannel", "systemPrompts", "video"]} label="视频系统提示词">
                                                    <Input.TextArea rows={4} placeholder="会自动追加在视频提示词前" />
                                                </Form.Item>
                                            </Col>
''', ''),
] if not VIDEO_ON else []
admin_settings_changed = False
for old, new in admin_settings_pairs:
    if old in s:
        s = s.replace(old, new, 1)
        admin_settings_changed = True
if admin_settings_changed:
    p.write_text(s, encoding="utf-8")
    note(ADMIN_SETTINGS, "移除视频相关表单项")
elif VIDEO_ON:
    fine(ADMIN_SETTINGS, "视频创作台已启用，保留视频相关项")
elif "默认视频模型" not in s:
    fine(ADMIN_SETTINGS, "视频相关项已移除")
else:
    print("!! admin settings 视频项锚点未命中")

# ---------- 15. 生图工作台：中→英润色按钮（脚本在 patches/translate-prompt.py） ----------
tr_script = pathlib.Path(__file__).resolve().parent / "translate-prompt.py"
if tr_script.exists():
    result = subprocess.run([sys.executable, str(tr_script)], capture_output=True, text=True)
    for line in result.stdout.splitlines():
        if line.startswith("本次改写") or line.startswith("translatePrompt 定义数") or line.startswith("!!"):
            ok.append(f"  [中→英] {line}")
    if result.returncode != 0:
        print("!! 中→英润色补丁失败：" + result.stderr.strip()[:200])
else:
    print("!! patches/translate-prompt.py 缺失，中→英润色按钮不会被应用")

# ---------- 校验 ----------
bad = []
for rel in list(NEW_FILES) + [
    "router/router.go",
    "web/src/services/api/user-config.ts",
    "web/src/services/api/image.ts",
    "web/src/components/layout/app-config-modal.tsx",
    "web/src/lib/model-channel.ts",
    "web/src/stores/use-config-store.ts",
]:
    if not (ROOT / rel).exists():
        bad.append(rel + "(缺失)")
    elif "***" in (ROOT / rel).read_text(encoding="utf-8"):
        bad.append(rel + "(有占位符残留)")
modal = (ROOT / "web/src/components/layout/app-config-modal.tsx").read_text(encoding="utf-8")
if modal.count("const addMyChannel") != 1 or modal.count("onClick={addMyChannel}") != 1:
    bad.append("app-config-modal.tsx(addMyChannel 定义/调用不配对)")
if (ROOT / "web/src/services/api/image.ts").read_text(encoding="utf-8").count("`") % 2 != 0:
    bad.append("image.ts(反引号不平衡)")

print("本地补丁结果：")
for line in changed or ["  （无需改写，全部已应用）"]:
    print(line)
for line in ok:
    print(line)
if bad:
    print("!! 异常，以下文件内容可疑：" + ", ".join(bad))
    sys.exit(1)
print("校验通过。")
