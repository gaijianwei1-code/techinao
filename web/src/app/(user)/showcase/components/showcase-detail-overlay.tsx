"use client";

import { ArrowLeft, Clock, Copy, Eye, Film, Heart, Spline, X } from "lucide-react";
import { Button } from "antd";

import type { ShowcaseWorkDetail } from "@/services/api/showcase";

function formatDate(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return `${date.getMonth() + 1}月${date.getDate()}日`;
}

function formatCount(value: number) {
    return value >= 1000 ? `${(value / 1000).toFixed(1)}k` : String(value);
}

// ShowcaseDetailOverlay 全屏作品预览弹层：视频预览 + 信息栏 + 创作过程入口。
export function ShowcaseDetailOverlay({
    work,
    onLike,
    onFork,
    onOpenProcess,
    onClose,
}: {
    work: ShowcaseWorkDetail;
    onLike: () => void;
    onFork: () => void;
    onOpenProcess: () => void;
    onClose: () => void;
}) {
    const project = parseProjectData(work.projectData);
    const nodeCount = project?.nodes?.length ?? 0;
    const connectionCount = project?.connections?.length ?? 0;

    return (
        <section className="fixed inset-0 z-50 flex flex-col bg-background">
            <header className="flex h-14 shrink-0 items-center justify-between border-b border-stone-200 pr-2 pl-1 dark:border-stone-800">
                <div className="flex min-w-0 items-center gap-2">
                    <Button type="text" icon={<ArrowLeft className="size-4" />} onClick={onClose} title="返回列表" />
                    <h2 className="truncate text-[15px] font-semibold">{work.title}</h2>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                    <Button type="primary" icon={<Copy className="size-3.5" />} onClick={onFork}>
                        复制项目
                    </Button>
                    <Button type="text" icon={<X className="size-4" />} onClick={onClose} title="关闭" />
                </div>
            </header>

            <div className="grid min-h-0 flex-1 gap-6 overflow-auto p-6 lg:grid-cols-[minmax(0,1fr)_360px]">
                <div className="min-w-0 self-start overflow-hidden rounded-2xl bg-black">
                    {work.preview ? (
                        <video src={work.preview} poster={work.cover || undefined} controls playsInline className="max-h-[calc(100dvh-8.5rem)] w-full" />
                    ) : work.cover ? (
                        <img src={work.cover} alt={work.title} className="w-full" />
                    ) : (
                        <div className="flex h-72 items-center justify-center text-stone-500">
                            <Film className="size-10" />
                        </div>
                    )}
                </div>

                <aside className="flex min-w-0 flex-col gap-4">
                    <div className="flex items-center gap-2.5">
                        {work.authorAvatar ? (
                            <img src={work.authorAvatar} alt={work.authorName} className="size-9 shrink-0 rounded-full object-cover" />
                        ) : (
                            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-stone-700 text-sm font-semibold text-white">
                                {work.authorName.slice(0, 1)}
                            </span>
                        )}
                        <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-semibold">{work.authorName}</p>
                            <p className="text-xs text-stone-500">
                                发布于 {formatDate(work.createdAt)} · {work.category}
                            </p>
                        </div>
                        <Button size="small" shape="round" icon={<Heart className={`size-3.5 ${work.liked ? "fill-red-500 text-red-500" : ""}`} />} onClick={onLike}>
                            {work.likes}
                        </Button>
                    </div>

                    <div className="flex items-center gap-4 text-xs text-stone-500">
                        <span className="inline-flex items-center gap-1">
                            <Eye className="size-3.5" />
                            {formatCount(nodeCount)} 节点
                        </span>
                        <span className="inline-flex items-center gap-1">
                            <Spline className="size-3.5" />
                            {formatCount(connectionCount)} 连线
                        </span>
                        <span className="inline-flex items-center gap-1">
                            <Clock className="size-3.5" />
                            {formatDate(work.createdAt)}
                        </span>
                    </div>

                    {work.description ? <p className="text-[13px] whitespace-pre-wrap text-stone-600 dark:text-stone-300">{work.description}</p> : null}

                    <div className="h-px bg-stone-200 dark:bg-stone-800" />

                    <button
                        type="button"
                        className="flex items-center gap-3 rounded-xl border border-stone-300 px-3.5 py-3 text-left transition hover:bg-stone-100 dark:border-stone-700 dark:hover:bg-stone-800/60"
                        onClick={onOpenProcess}
                    >
                        <Spline className="size-5 shrink-0" />
                        <span className="min-w-0 flex-1">
                            <span className="block text-[13px] font-semibold">查看创作过程</span>
                            <span className="block text-[11px] text-stone-500">节点画布 · 连线 · 生成记录</span>
                        </span>
                        <ArrowLeft className="size-4 shrink-0 rotate-180 text-stone-500" />
                    </button>
                </aside>
            </div>
        </section>
    );
}

export function parseProjectData(projectData: string): { nodes: { id: string; type: string }[]; connections: unknown[] } | null {
    try {
        const parsed = JSON.parse(projectData);
        if (parsed && Array.isArray(parsed.nodes)) return parsed;
        return null;
    } catch {
        return null;
    }
}
