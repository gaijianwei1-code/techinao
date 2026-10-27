#!/usr/bin/env python3
"""计算并删除「我的画布 / 视频创作台」相关的路由与独占模块。

策略：从所有保留的入口（app/**/page.tsx、layout.tsx、route.ts 等）出发做可达性分析，
canvas/ 与 video/ 目录下不可达的文件视为独占代码，连同路由一起删除。
被其它页面引用的公共件（如 asset-picker-modal）会被判为可达，自动保留。

用法：python3 patches/prune-canvas.py [--dry]
"""
import re
import shutil
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "web" / "src"

# 开关：patches/.video-enabled 存在时保留「视频创作台」（由 patches/video-feature.py on|off 维护）
VIDEO_ENABLED = (Path(__file__).resolve().parent / ".video-enabled").exists()

# 画布现在是产品核心功能（已完整移植），不再参与剪枝；本脚本只保留视频开关与广告轮播清理。
# 要整体删除的路由目录
ROUTE_DIRS = []
if not VIDEO_ENABLED:
    ROUTE_DIRS.append(SRC / "app/(user)/video")
ROUTE_FILES = [
    SRC / "app/(user)/home-banner-carousel.tsx",   # 首页广告轮播，一起移除
]
# 允许评估删除的目录（只在这里面删，别的目录一律不动）
CANDIDATE_DIRS = [] if VIDEO_ENABLED else [SRC / "app/(user)/video"]

EXTS = [".tsx", ".ts", ".jsx", ".js", ".mjs", ".css"]
IMPORT_RE = re.compile(
    r"""(?:^|[^\w.])import\s*(?:type\s*)?(?:[\s\S]*?from\s*)?["']([^"']+)["']"""
    r"""|(?:import|require)\s*\(\s*["']([^"']+)["']\s*\)"""
    r"""|export\s*(?:type\s*)?[\s\S]*?from\s*["']([^"']+)["']""",
    re.MULTILINE,
)


def resolve(spec: str, origin: Path):
    if spec.startswith("@/"):
        base = SRC / spec[2:]
    elif spec.startswith("."):
        base = (origin.parent / spec).resolve()
    else:
        return None
    if not str(base).startswith(str(SRC)):
        return None
    cands = [base] + [base.with_suffix(e) for e in EXTS]
    for e in EXTS:
        cands.append(base / f"index{e}")
    for c in cands:
        if c.is_file():
            return c.resolve()
    return None


def walk_all():
    for p in SRC.rglob("*"):
        if p.is_file() and p.suffix in EXTS:
            yield p.resolve()


def main():
    dry = "--dry" in sys.argv
    if not dry:
        for d in ROUTE_DIRS:
            if d.exists():
                shutil.rmtree(d)
        for f in ROUTE_FILES:
            if f.exists():
                f.unlink()

    files = list(walk_all())
    specs = {}
    for f in files:
        try:
            text = f.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        out = []
        for m in IMPORT_RE.finditer(text):
            spec = m.group(1) or m.group(2) or m.group(3)
            if spec:
                target = resolve(spec, f)
                if target:
                    out.append(target)
        specs[f] = out

    # 入口：所有路由/布局/中间件（canvas 已是核心功能，同样作为入口参与可达性分析）
    roots = []
    for f in files:
        rel = str(f)
        if not VIDEO_ENABLED and rel.startswith(str(SRC / "app/(user)/video")):
            continue
        name = f.name
        if name in {"page.tsx", "layout.tsx", "route.ts", "template.tsx", "default.tsx", "not-found.tsx", "middleware.ts", "globals.css", "app-providers.tsx"}:
            roots.append(f)
    if not roots:
        print("!! 找不到入口文件，中止")
        return 1

    reachable = set()
    deleted = set()
    # 迭代：先按可达性找出 canvas/video 独占文件，再收敛"引用了被删文件、但自己也没人要"的死代码
    # （tsconfig 会检查不在依赖图里的文件，留着会直接让 next build 因找不到模块而失败）
    while True:
        reachable = set()
        stack = [r for r in roots if r not in deleted]
        while stack:
            cur = stack.pop()
            if cur in reachable or cur in deleted:
                continue
            reachable.add(cur)
            for nxt in specs.get(cur, []):
                if nxt not in reachable and nxt not in deleted:
                    stack.append(nxt)

        new_deleted = set()
        for f in files:
            if f in deleted or f in reachable:
                continue
            if any(str(f).startswith(str(d)) for d in CANDIDATE_DIRS):
                new_deleted.add(f)          # canvas/video 里的独占文件
            elif any(dep in deleted for dep in specs.get(f, [])):
                new_deleted.add(f)          # 引用了已删文件的死代码
        if not new_deleted:
            break
        deleted |= new_deleted

    orphan = sorted(deleted)

    print(f"入口数: {len(roots)}  可达文件数: {len(reachable)}")
    print(f"将被删除的独占文件: {len(orphan)} 个")
    for f in sorted(orphan):
        print("  -", f.relative_to(ROOT))
    if dry:
        print("（dry-run，未实际删除）")
        return 0

    for f in orphan:
        f.unlink()
    # 清理空目录
    for d in CANDIDATE_DIRS:
        if d.exists():
            for sub in sorted([p for p in d.rglob("*") if p.is_dir()], key=lambda p: -len(p.parts)):
                if not any(sub.iterdir()):
                    sub.rmdir()
    return 0


if __name__ == "__main__":
    sys.exit(main())
