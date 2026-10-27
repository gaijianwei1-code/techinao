#!/usr/bin/env python3
"""视频创作台开关。

用法：
    python3 patches/video-feature.py status   # 看当前状态（默认）
    python3 patches/video-feature.py on       # 把视频创作台加回来
    python3 patches/video-feature.py off      # 再收回去（回到只有生图/提示词/素材的状态）

背景：视频创作台当初被剪掉时，实际只删了 web/src/app/(user)/video/ 下 2 个文件
（page.tsx、components/kling-v26-workbench-panel.tsx）——后端接口和视频相关 lib 都还在。
原始文件备份在 patches/video-files/ 下。

开关负责四处改动：
  1. 增删那 2 个页面文件
  2. 导航项 web/src/constant/navigation-tools.ts（连同 patches/files 里的存档副本一起改）
  3. 首页卡片 web/src/app/(user)/page.tsx（连同存档副本一起改）
  4. 配置项：配置弹窗「默认视频模型」+ 后台「默认视频模型 / 视频系统提示词」

并维护开关文件 patches/.video-enabled：
  - prune-canvas.py 看到它就放过 video 目录（否则 ./start.sh rebuild 时又被剪掉）
  - apply.py 第 13/14 节看到它就跳过视频相关的删除

改完记得重建：./start.sh rebuild
"""
import pathlib
import re
import shutil
import sys

PATCHES = pathlib.Path(__file__).resolve().parent
ROOT = PATCHES.parent
MARKER = PATCHES / ".video-enabled"
FILES = PATCHES / "video-files"

NAV = ROOT / "web/src/constant/navigation-tools.ts"
HOME = ROOT / "web/src/app/(user)/page.tsx"
MODAL = ROOT / "web/src/components/layout/app-config-modal.tsx"
ADMIN = ROOT / "web/src/app/(admin)/admin/settings/page.tsx"
STORED_NAV = PATCHES / "files/web/src/constant/navigation-tools.ts"
STORED_HOME = PATCHES / "files/web/src/app/(user)/page.tsx"

VIDEO_RELS = [
    "web/src/app/(user)/video/page.tsx",
    "web/src/app/(user)/video/components/kling-v26-workbench-panel.tsx",
]

NAV_ANCHOR = '''    {
        slug: "image",
        label: "生图工作台",
        icon: ImagePlus,
    },
'''
NAV_ENTRY = '''    {
        slug: "video",
        label: "视频创作台",
        icon: Video,
    },
'''

HOME_ANCHOR = '    { href: "/image", title: "生图工作台", desc: "写提示词、带参考图，直接出图", icon: ImagePlus },\n'
HOME_CARD = '    { href: "/video", title: "视频创作台", desc: "文生视频 / 图生视频，带首尾帧与运镜设置", icon: Video },\n'

MODAL_ANCHOR = '    { capability: "image", modelKey: "imageModel", channelKey: "imageChannelId", modelsKey: "imageModels", defaultLabel: "默认生图模型", optionsLabel: "生图模型可选项" },\n'
MODAL_GROUP = '    { capability: "video", modelKey: "videoModel", channelKey: "videoChannelId", modelsKey: "videoModels", defaultLabel: "默认视频模型", optionsLabel: "视频模型可选项" },\n'
MODAL_DESC_OFF = "可为生图和文本分别选择不同渠道的模型。"
MODAL_DESC_ON = "可为生图、视频和文本分别选择不同渠道的模型。"

ADMIN_MODEL_ANCHOR = '''                                    <Col xs={24} md={6}>
                                        <Form.Item name={["public", "modelChannel", "defaultTextModel"]} label="默认文本模型">
                                            <Select showSearch={{ optionFilterProp: ["label", "value"] }} allowClear options={publicModels.map((item) => ({ label: publicModelLabel(item), value: item }))} />
                                        </Form.Item>
                                    </Col>
'''
ADMIN_MODEL_COL = '''                                    <Col xs={24} md={6}>
                                        <Form.Item name={["public", "modelChannel", "defaultVideoModel"]} label="默认视频模型">
                                            <Select showSearch={{ optionFilterProp: ["label", "value"] }} allowClear options={publicModels.map((item) => ({ label: publicModelLabel(item), value: item }))} />
                                        </Form.Item>
                                    </Col>
'''

ADMIN_PROMPT_ANCHOR = '''                                            <Col xs={24} md={12}>
                                                <Form.Item name={["public", "modelChannel", "systemPrompts", "image"]} label="生图系统提示词">
                                                    <Input.TextArea rows={4} placeholder="会自动追加在生图提示词前，不在输入框中展示" />
                                                </Form.Item>
                                            </Col>
'''
ADMIN_PROMPT_COL = '''                                            <Col xs={24} md={12}>
                                                <Form.Item name={["public", "modelChannel", "systemPrompts", "video"]} label="视频系统提示词">
                                                    <Input.TextArea rows={4} placeholder="会自动追加在视频提示词前" />
                                                </Form.Item>
                                            </Col>
'''

changes = []
problems = []


def read(p):
    return p.read_text(encoding="utf-8")


def write(p, s):
    p.write_text(s, encoding="utf-8")


def import_line_bounds(s):
    """返回 lucide-react 那行 import 的 (起, 止) 下标。"""
    marker = 'from "lucide-react";'
    i = s.find(marker)
    if i < 0:
        return None
    start = s.rfind("import {", 0, i)
    if start < 0:
        return None
    return start, i


def add_icon(s, icon):
    bounds = import_line_bounds(s)
    if not bounds:
        return s, False
    start, end = bounds
    line = s[start:end]
    if re.search(rf"\b{icon}\b", line):
        return s, False
    if " }" in line:                     # 'import { A, B } ' → 'import { A, B, C } '
        new_line = line.replace(" }", f", {icon} }}", 1)
    else:                                # 'import { A, B}' → 'import { A, B, C}'
        new_line = line.replace("}", f", {icon}}}", 1)
    return s[:start] + new_line + s[end:], True


def del_icon(s, icon):
    bounds = import_line_bounds(s)
    if not bounds:
        return s, False
    start, end = bounds
    line = s[start:end]
    if not re.search(rf"\b{icon}\b", line):
        return s, False
    for pat, rep in ((f", {icon} }}", " }"), (f"{{{icon}, ", "{"), (f", {icon},", ","), (f"{{{icon} }}", "{}")):
        if pat in line:
            return s[:start] + line.replace(pat, rep, 1) + s[end:], True
    return s, False


def ensure_entry(path, enable, kind):
    """kind: nav / home —— 增删一条入口（含 lucide 图标名）。"""
    if not path.exists():
        problems.append(f"{path.relative_to(ROOT) if ROOT in path.parents else path} 不存在")
        return
    s = read(path)
    anchor = NAV_ANCHOR if kind == "nav" else HOME_ANCHOR
    entry = NAV_ENTRY if kind == "nav" else HOME_CARD
    has = ('slug: "video"' in s) if kind == "nav" else ('href: "/video"' in s)
    if enable and not has:
        if anchor not in s:
            problems.append(f"{path.name}: 找不到插入锚点")
            return
        s = s.replace(anchor, anchor + entry, 1)
        s, _ = add_icon(s, "Video")
        write(path, s)
        changes.append(f"{path.name}: 加回视频入口")
    elif not enable and has:
        s = s.replace(entry, "", 1)
        s, _ = del_icon(s, "Video")
        write(path, s)
        changes.append(f"{path.name}: 移除视频入口")


def ensure_modal(enable):
    s = read(MODAL)
    has = 'capability: "video"' in s
    if enable and not has:
        if MODAL_ANCHOR not in s:
            problems.append("app-config-modal.tsx: 找不到 modelGroups 锚点")
            return
        s = s.replace(MODAL_ANCHOR, MODAL_ANCHOR + MODAL_GROUP, 1)
        if MODAL_DESC_OFF in s:
            s = s.replace(MODAL_DESC_OFF, MODAL_DESC_ON, 1)
        write(MODAL, s)
        changes.append("app-config-modal.tsx: 加回「默认视频模型」")
    elif not enable and has:
        s = s.replace(MODAL_GROUP, "", 1)
        if MODAL_DESC_ON in s:
            s = s.replace(MODAL_DESC_ON, MODAL_DESC_OFF, 1)
        write(MODAL, s)
        changes.append("app-config-modal.tsx: 移除「默认视频模型」")


def ensure_admin(enable):
    s = read(ADMIN)
    has = '"defaultVideoModel"' in s
    if enable and not has:
        if ADMIN_MODEL_ANCHOR not in s or ADMIN_PROMPT_ANCHOR not in s:
            problems.append("admin/settings/page.tsx: 找不到插入锚点")
            return
        s = s.replace(ADMIN_MODEL_ANCHOR, ADMIN_MODEL_ANCHOR + ADMIN_MODEL_COL, 1)
        s = s.replace(ADMIN_PROMPT_ANCHOR, ADMIN_PROMPT_ANCHOR + ADMIN_PROMPT_COL, 1)
        write(ADMIN, s)
        changes.append("admin/settings/page.tsx: 加回默认视频模型 + 视频系统提示词")
    elif not enable and has:
        s = s.replace(ADMIN_MODEL_COL, "", 1).replace(ADMIN_PROMPT_COL, "", 1)
        write(ADMIN, s)
        changes.append("admin/settings/page.tsx: 移除视频相关表单项")


def ensure_files(enable):
    for rel in VIDEO_RELS:
        dst = ROOT / rel
        src = FILES / rel
        if enable:
            if not src.exists():
                problems.append(f"补丁源文件缺失：{src.relative_to(PATCHES)}")
                continue
            if dst.exists() and dst.read_bytes() == src.read_bytes():
                continue
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(src, dst)
            changes.append(f"{rel}: 已还原（{dst.stat().st_size} 字节）")
        else:
            if dst.exists():
                dst.unlink()
                changes.append(f"{rel}: 已删除")
    video_dir = ROOT / "web/src/app/(user)/video"
    if not enable and video_dir.exists():
        try:
            shutil.rmtree(video_dir)
        except OSError as exc:
            problems.append(f"清目录失败 {video_dir}: {exc}")


def current_state():
    on_files = all((ROOT / rel).exists() for rel in VIDEO_RELS)
    on_nav = NAV.exists() and 'slug: "video"' in read(NAV)
    on_home = HOME.exists() and 'href: "/video"' in read(HOME)
    on_modal = MODAL.exists() and 'capability: "video"' in read(MODAL)
    on_admin = ADMIN.exists() and '"defaultVideoModel"' in read(ADMIN)
    return {
        "页面文件": on_files,
        "导航入口": on_nav,
        "首页卡片": on_home,
        "配置项(弹窗)": on_modal,
        "配置项(后台)": on_admin,
        "开关文件": MARKER.exists(),
    }


def main():
    action = (sys.argv[1] if len(sys.argv) > 1 else "status").lower()
    if action not in ("on", "off", "status"):
        print(__doc__)
        return 2

    if action == "status":
        print("视频创作台当前状态：")
        for key, value in current_state().items():
            print(f"  {key:14s} {'✔ 已启用' if value else '✘ 未启用'}")
        print("\n切换：python3 patches/video-feature.py on|off   （改完跑 ./start.sh rebuild）")
        return 0

    enable = action == "on"
    ensure_files(enable)
    ensure_entry(NAV, enable, "nav")
    ensure_entry(STORED_NAV, enable, "nav")
    ensure_entry(HOME, enable, "home")
    ensure_entry(STORED_HOME, enable, "home")
    ensure_modal(enable)
    ensure_admin(enable)

    if enable:
        MARKER.write_text("视频创作台已启用：prune-canvas.py 放过 video 目录，apply.py 跳过视频相关删除。\n", encoding="utf-8")
    elif MARKER.exists():
        MARKER.unlink()

    print(f"=== 视频创作台 {action} ===")
    if changes:
        for line in changes:
            print("  ✔", line)
    else:
        print("  （无需改动，已是目标状态）")
    for line in problems:
        print("  !!", line)
    print("\n当前状态：")
    for key, value in current_state().items():
        print(f"  {key:14s} {'✔ 已启用' if value else '✘ 未启用'}")
    if enable:
        print("\n下一步：./start.sh rebuild   （约 2-5 分钟）")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
