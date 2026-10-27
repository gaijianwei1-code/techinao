# 本地补丁说明（我自己的 AI 工作台 / infinite-canvas 定制版）

这些补丁是本机部署为了适配国内网络、去掉用不上的功能、以及换成自己的品牌而加的。
**它们是幂等的**：`python3 patches/apply.py` 可以反复执行，已打过的会跳过，不会重复改。

`./start.sh rebuild` 和 `./start.sh update` 会自动调用它，所以升级源码后补丁不会丢。

## 项目位置

- **Windows**：`D:\.Hermes\infinite-canvas`（资源管理器可见）＝ `/mnt/d/.Hermes/infinite-canvas`（WSL 里的路径）
- **macOS**：`~/infinite-canvas`（迁移步骤见 `docs/overview/migrate-to-mac.md`）

源码、`data/` 数据库、补丁脚本、启动脚本都在这一份里。补丁脚本本身与平台无关，但需要宿主机有 `python3`（macOS 上 `xcode-select --install` 或 `brew install python3`）。

## 怎么跑

```bash
cd /mnt/d/.Hermes/infinite-canvas   # macOS 上换成 cd ~/infinite-canvas
python3 patches/apply.py        # 检查/重放全部补丁（幂等）
./start.sh rebuild              # 重建镜像并重启（会自动先跑 apply.py）
python3 patches/prune-canvas.py --dry   # 只看「我的画布/视频创作台」会删掉哪些文件

python3 patches/video-feature.py status # 看「视频创作台」开关状态
python3 patches/video-feature.py on     # 把视频创作台加回来（改完跑 ./start.sh rebuild）
python3 patches/video-feature.py off    # 再收回去
```

## 视频创作台开关（patches/video-feature.py）

当初剪掉「视频创作台」时实际只删了 `web/src/app/(user)/video/` 下 2 个文件，原文件备份在
`patches/video-files/` 下；后端视频接口（`/api/v1/videos`、`/video-tasks`、`/generation-logs/videos`）
和视频相关 lib（`services/api/video.ts`、`lib/*-video.ts`、`components/video-settings-panel.tsx`）一直没动。

开关负责四处：① 增删那 2 个页面文件 ② 导航项 ③ 首页卡片 ④ 配置项（配置弹窗「默认视频模型」+
后台「默认视频模型 / 视频系统提示词」），并维护开关文件 `patches/.video-enabled`：
`prune-canvas.py` 看到它就放过 video 目录，`apply.py` 第 13/14 节看到它就跳过视频相关的删除 ——
所以开了开关以后，`./start.sh rebuild` / `update` 都不会再把它剪掉。

## 补丁清单

### 一、国内网络

| 文件 | 内容 | 为什么 |
|---|---|---|
| `Dockerfile` | build 阶段 `ENV GOPROXY=https://goproxy.cn,direct`、`GOSUMDB=sum.golang.google.cn` | 容器内 `proxy.golang.org` 不可达，`go build` 会 i/o timeout |
| `Dockerfile` | web-build stage 加 `ENV BUN_CONFIG_REGISTRY=https://registry.npmmirror.com` | `registry.npmjs.org` 大陆直连会超时，`bun install` 抽 `next` 包直接失败 |
| `Dockerfile` | 运行时 stage 的 Debian 源换 `mirrors.aliyun.com` | `deb.debian.org` 拉 Packages 会卡到超时 |
| `service/prompt_fetch.go` | 7 个提示词源统一加镜像前缀，默认 `https://ghfast.top`（`PROMPT_RAW_MIRROR` 可覆盖，置空则直连）；拉取超时 30s → 120s | `raw.githubusercontent.com` 国内直连超时，提示词库爬不下来 |
| `service/prompt_cover.go`（新增）、`handler/prompts.go`、`router/router.go`、`service/prompt_fetch.go` | 同步 YouMind 两个分类时把封面图抓到 `data/prompts/covers/`，封面地址改写为 `/api/prompt-covers/<hash>`（`GET /api/prompt-covers/:name` 由后端返回图片）；抓取失败退回图片镜像，默认 `https://wsrv.nl/?url=`（`PROMPT_IMAGE_MIRROR` 可覆盖，置空则只走直连） | `cms-assets.youmind.com` 国内直连不通（TCP 443 超时），提示词库卡片整片没有预览图 |

### 二、拉取模型列表走服务端（修 CORS）

| 文件 | 内容 |
|---|---|
| `handler/channel_models.go`（新增） | `POST /api/v1/channel-models`，登录可用 |
| `service/channel_models.go`（新增） | 服务端请求 `<baseUrl>/models`，走 `SafeProxyHTTPClient`（禁内网，防 SSRF） |
| `router/router.go` | 在 `/api/v1` 组注册该路由 |
| `web/src/services/api/image.ts` | `fetchImageModels()` 登录后优先走服务端；仅当后端路由缺失时才退回浏览器直连 |
| `web/src/services/api/user-config.ts` | 前端调用封装 |

上游（如 aiaaa.cc）不返回 `Access-Control-Allow-Origin`、`OPTIONS` 预检 403，浏览器直连必然失败，必须由服务端代拉。

### 三、去掉不用的功能（我的画布 / 视频创作台）

| 动作 | 说明 |
|---|---|
| `patches/prune-canvas.py` | 从所有保留路由做**可达性分析**，删掉 `(user)/canvas`、`(user)/video` 下无人引用的文件，并收敛「引用了被删文件、但自己也没人要」的死代码 |
| 删除的路由 | `/canvas`、`/canvas/[id]`、`/video`（现返回 404） |
| 删除的组件 | `home-banner-carousel`（首页广告轮播）、`github-link`、`version-release-modal`、`use-version-check`、旧 `favicon.ico` |
| 保留的公共件 | `asset-picker-modal.tsx`、`use-canvas-store.ts`、`canvas/types.ts` —— 生图页、工作流页、素材同步仍在用，删了会编译不过 |
| `web/src/constant/navigation-tools.ts` | 顶部/手机导航只留：生图工作台、提示词库、我的素材 |

### 四、品牌与个性化

| 文件 | 内容 |
|---|---|
| `web/src/app/(user)/page.tsx` | 首页重写为自用工作台导航页（原来的项目列表 + 广告轮播 + 画布助手全部去掉） |
| `web/public/logo.svg` | 新 logo：圆角方框 + 四角星（用作 mask，纯黑白形状） |
| `web/src/app/icon.svg` | 新站点图标（替代上游 `favicon.ico`） |
| `web/src/app/layout.tsx` | 浏览器标题/描述 → 「我自己的 AI 工作台」 |
| `web/src/components/layout/app-top-nav.tsx`、`(user)/login/page.tsx`、`(admin)/admin/layout.tsx` | 品牌文案同步替换 |
| `web/src/components/layout/app-top-nav.tsx` | 登录页不再显示顶部工具栏（原来的 `hideHeader` 只判断已删除的画布详情页，所以登录页会露出工具栏） |
| `web/src/components/layout/user-status-actions.tsx` | 去掉右上角 GitHub 图标与版本弹窗 |
| `web/src/app/(admin)/admin/layout.tsx` | 后台本来只有「前往画布」按钮（`/canvas`，新窗口打开），画布删掉后成了 404 死链；改为三处「**返回主页**」入口：侧边栏底部按钮、右上角按钮、品牌区可点（都是同窗跳转 `/`） |

品牌文件放在 `patches/files/` 下（按原路径存放），`apply.py` 会整文件覆盖回去；整文件新增的后端文件（如 `service/prompt_cover.go`）也走这里。

### 五、配置与用户偏好定制

| 文件 | 内容 |
|---|---|
| `web/src/lib/model-channel.ts` | 渠道预设去掉上游**返利链接**（metaso/88api/apimart/mimo 的 `?aff=/ref=` 尾巴）；OpenAI 标签改「OpenAI 兼容」 |
| `web/src/lib/model-channel.test.ts` | 同步更新断言，`bun test` 不红 |
| `web/src/stores/use-config-store.ts` | 默认 Base URL 指向 `https://aiaaa.cc/v1`（新浏览器打开就是自己的中转） |
| `web/src/components/layout/app-config-modal.tsx` | 副标题改「模型、渠道与出图默认行为 · 配置只存在本机」；新增「**我的中转**」一键按钮（自动填好 `aiaaa.cc/v1` + `gpt-image-2`）；「新增渠道」默认沿用你填的 Base URL；「画布默认生图张数」文案改为「默认生图张数」 |
| `web/src/components/layout/app-config-modal.tsx` | 第二轮：删掉「默认视频模型」「默认音频模型」两个下拉（保存校验同步放宽）；「渠道模式」切换（本地直连/云端渠道）换成「**中转站推荐**」卡片，按钮指向 `https://aiaaa.cc/register?aff=FR4MTDLX9AB2`；工作台固定本地直连模式 |
| `web/src/app/(admin)/admin/settings/page.tsx` | 后台同步去掉已废弃的「默认视频模型」「视频系统提示词」表单项 |
| `web/src/app/(user)/image/page.tsx` | 「**中→英润色**」按钮（侧边工作台带文字、悬浮布局是图标）：调文本模型把中文想法改写成英文生图提示词填回输入框，可一键还原；未配置文本模型时给明确提示。补丁脚本：`patches/translate-prompt.py` |

## 数据

- SQLite：`~/infinite-canvas/data/infinite-canvas.db`
- 提示词库：`prompts` 表，首次同步后约 1600 条（7 个分类）；后台「提示词管理」有同步按钮，默认每天 0 点定时同步
- 提示词封面：`data/prompts/covers/`，同步时抓到本地（YouMind 两个分类经图片镜像抓取，约 250 张、一次约 1 分钟），之后由后端直接返回，不再依赖外网

## 回退

删掉 `patches/` 目录，再重新下载上游源码覆盖：

```bash
curl -fsSL -o /tmp/ic.tar.gz "https://ghfast.top/https://github.com/tigerowo/infinite-canvas/archive/refs/heads/main.tar.gz"
tar xzf /tmp/ic.tar.gz -C ~/infinite-canvas --strip-components=1 && cd ~/infinite-canvas && ./start.sh rebuild
```
