"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Copy, Film, Heart, Search, Spline, X } from "lucide-react";
import { App, Input, Spin } from "antd";

import { ProcessCanvas } from "@/app/(user)/showcase/components/process-canvas";
import { ShowcaseDetailOverlay, parseProjectData } from "@/app/(user)/showcase/components/showcase-detail-overlay";
import { useCanvasStore } from "@/app/(user)/canvas/stores/use-canvas-store";
import { forkShowcaseWork, getShowcaseWork, likeShowcaseWork, listShowcaseWorks, type ShowcaseWorkDetail, type ShowcaseWorkSummary } from "@/services/api/showcase";
import { useUserStore } from "@/stores/use-user-store";

function isVideoUrl(url: string) {
    return /\.(mp4|webm|mov|m4v)(\?|#|$)/i.test(url);
}

function WorkCover({ src, alt }: { src: string; alt: string }) {
    if (!src) {
        return (
            <div className="flex h-full items-center justify-center text-stone-600">
                <Film className="size-8" />
            </div>
        );
    }
    if (isVideoUrl(src)) {
        return <video src={src} muted preload="metadata" playsInline className="h-full w-full object-cover" />;
    }
    return <img src={src} alt={alt} loading="lazy" className="h-full w-full object-cover" />;
}

// 创作精选：浏览他人发布的画布作品，预览成片并查看创作过程。
export default function ShowcasePage() {
    const { message } = App.useApp();
    const router = useRouter();
    const [works, setWorks] = useState<ShowcaseWorkSummary[]>([]);
    const [categories, setCategories] = useState<string[]>(["全部"]);
    const [category, setCategory] = useState("全部");
    const [keyword, setKeyword] = useState("");
    const [loading, setLoading] = useState(true);
    const [detail, setDetail] = useState<ShowcaseWorkDetail | null>(null);
    const [mode, setMode] = useState<"detail" | "process">("detail");

    const loadWorks = useCallback(async (nextCategory: string, nextKeyword: string) => {
        setLoading(true);
        try {
            const result = await listShowcaseWorks({ category: nextCategory === "全部" ? "" : nextCategory, keyword: nextKeyword, pageSize: 50 });
            setWorks(result.items || []);
            if (result.categories?.length) {
                setCategories((current) => (current.length > 1 ? current : ["全部", ...result.categories]));
            }
        } catch (error) {
            message.error(error instanceof Error ? error.message : "加载创作精选失败");
        } finally {
            setLoading(false);
        }
    }, [message]);

    useEffect(() => {
        void loadWorks(category, keyword);
    }, [category, keyword, loadWorks]);

    const likeWork = useCallback(async (id: string) => {
        if (!useUserStore.getState().token) {
            message.warning("请先登录后再点赞");
            return;
        }
        try {
            const result = await likeShowcaseWork(id);
            setWorks((current) => current.map((work) => (work.id === id ? { ...work, liked: result.liked, likes: result.likes } : work)));
            setDetail((current) => (current && current.id === id ? { ...current, liked: result.liked, likes: result.likes } : current));
        } catch (error) {
            message.error(error instanceof Error ? error.message : "点赞失败");
        }
    }, [message]);

    const forkWork = useCallback(async (id: string) => {
        if (!useUserStore.getState().token) {
            message.warning("请先登录后再复制项目");
            return;
        }
        try {
            message.loading({ content: "正在复制到我的画布…", key: id, duration: 0 });
            const result = await forkShowcaseWork(id);
            const newId = useCanvasStore.getState().importProject(result.data);
            message.success({ content: "已复制到我的画布", key: id });
            router.push(`/canvas/${newId}`);
        } catch (error) {
            message.error({ content: error instanceof Error ? error.message : "复制失败", key: id });
        }
    }, [message, router]);

    const openDetail = useCallback(async (id: string, nextMode: "detail" | "process" = "detail") => {
        try {
            const work = await getShowcaseWork(id);
            setDetail(work);
            setMode(nextMode);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "作品加载失败");
        }
    }, [message]);

    const closeOverlays = useCallback(() => {
        setDetail(null);
        setMode("detail");
    }, []);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key !== "Escape") return;
            if (mode === "process") setMode("detail");
            else if (detail) closeOverlays();
        };
        document.addEventListener("keydown", onKeyDown);
        return () => document.removeEventListener("keydown", onKeyDown);
    }, [detail, mode, closeOverlays]);

    const processProject = useMemo(() => {
        if (!detail || mode !== "process") return null;
        const parsed = parseProjectData(detail.projectData);
        if (!parsed) return null;
        return parsed as unknown as { nodes: Parameters<typeof ProcessCanvas>[0]["nodes"]; connections: Parameters<typeof ProcessCanvas>[0]["connections"] };
    }, [detail, mode]);

    return (
        <main className="mx-auto h-full max-w-7xl overflow-y-auto px-6 py-8">
            <div>
                <h1 className="text-2xl font-bold tracking-tight">创作精选</h1>
                <p className="mt-1 text-[13px] text-stone-500">看看别人用画布创作了什么，打开即可查看完整创作过程</p>
            </div>

            <div className="mt-6 flex flex-wrap items-center justify-between gap-4">
                <nav className="hide-scrollbar flex items-center gap-1 overflow-x-auto">
                    {categories.map((item) => (
                        <button
                            key={item}
                            type="button"
                            className={`relative shrink-0 rounded-lg px-3 py-2 text-[13px] transition ${
                                item === category ? "font-semibold text-stone-950 dark:text-stone-100" : "text-stone-500 hover:text-stone-950 dark:text-stone-400 dark:hover:text-stone-100"
                            }`}
                            onClick={() => setCategory(item)}
                        >
                            {item}
                            {item === category ? <span className="absolute inset-x-1/2 bottom-1 h-0.5 w-4 -translate-x-1/2 rounded-full bg-stone-950 dark:bg-stone-100" /> : null}
                        </button>
                    ))}
                </nav>
                <Input
                    allowClear
                    prefix={<Search className="size-4 text-stone-400" />}
                    placeholder="搜索作品或作者"
                    className="w-60"
                    onChange={(event) => setKeyword(event.target.value.trim())}
                />
            </div>

            {loading && !works.length ? (
                <div className="flex h-64 items-center justify-center text-stone-500">
                    <Spin />
                </div>
            ) : works.length ? (
                <section className={`mt-5 grid gap-x-5 gap-y-6 transition-opacity sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 ${loading ? "opacity-40" : ""}`}>
                    {works.map((work) => (
                        <article key={work.id} className="group cursor-pointer" onClick={() => void openDetail(work.id)}>
                            <div className="relative aspect-video overflow-hidden rounded-xl bg-stone-100 dark:bg-stone-800">
                                <WorkCover src={work.cover} alt={work.title} />
                                <button
                                    type="button"
                                    className="absolute inset-0 m-auto flex h-fit w-fit items-center gap-1.5 rounded-full bg-stone-50 px-4 py-2 text-[13px] font-semibold text-stone-950 opacity-0 transition group-hover:opacity-100"
                                    onClick={(event) => {
                                        event.stopPropagation();
                                        void openDetail(work.id, "process");
                                    }}
                                >
                                    <Spline className="size-3.5" />
                                    查看创作过程
                                </button>
                                <button
                                    type="button"
                                    className={`absolute top-2.5 right-2.5 inline-flex items-center gap-1 rounded-full bg-stone-950/60 px-2.5 py-1 font-mono text-xs text-white backdrop-blur transition hover:bg-stone-950/80 ${
                                        work.liked ? "text-red-400" : ""
                                    }`}
                                    onClick={(event) => {
                                        event.stopPropagation();
                                        void likeWork(work.id);
                                    }}
                                >
                                    <Heart className={`size-3 ${work.liked ? "fill-red-400" : ""}`} />
                                    {work.likes}
                                </button>
                            </div>
                            <h3 className="mt-2.5 truncate text-sm font-semibold">{work.title}</h3>
                            <div className="mt-1.5 flex items-center gap-2 text-[13px] text-stone-500">
                                {work.authorAvatar ? (
                                    <img src={work.authorAvatar} alt={work.authorName} className="size-5 shrink-0 rounded-full object-cover" />
                                ) : (
                                    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-stone-600 text-[11px] font-semibold text-white">
                                        {work.authorName.slice(0, 1)}
                                    </span>
                                )}
                                <span className="truncate">{work.authorName}</span>
                                <span className="text-stone-400">·</span>
                                <span className="shrink-0">{work.category}</span>
                            </div>
                        </article>
                    ))}
                </section>
            ) : (
                <div className="py-24 text-center text-stone-500">
                    <Film className="mx-auto mb-3 size-9 text-stone-400" />
                    <p className="mb-4 text-sm">没有找到相关作品</p>
                    <button
                        type="button"
                        className="rounded-lg px-3 py-2 text-[13px] text-stone-500 transition hover:bg-stone-100 hover:text-stone-950 dark:hover:bg-stone-800 dark:hover:text-stone-100"
                        onClick={() => {
                            setKeyword("");
                            setCategory("全部");
                        }}
                    >
                        清除筛选条件
                    </button>
                </div>
            )}

            {detail && (mode === "detail" || !processProject) ? (
                <ShowcaseDetailOverlay
                    work={detail}
                    onLike={() => void likeWork(detail.id)}
                    onFork={() => void forkWork(detail.id)}
                    onOpenProcess={() => setMode("process")}
                    onClose={closeOverlays}
                />
            ) : null}

            {detail && mode === "process" && processProject ? (
                <section className="fixed inset-0 z-50 flex flex-col bg-background">
                    <header className="flex h-14 shrink-0 items-center justify-between border-b border-stone-200 pr-2 pl-1 dark:border-stone-800">
                        <div className="flex min-w-0 items-center gap-2">
                            <button
                                type="button"
                                className="flex size-8 items-center justify-center rounded-lg text-stone-500 transition hover:bg-stone-100 hover:text-stone-950 dark:hover:bg-stone-800 dark:hover:text-stone-100"
                                title="返回预览"
                                onClick={() => setMode("detail")}
                            >
                                <ArrowLeft className="size-4" />
                            </button>
                            <h2 className="truncate text-[15px] font-semibold">{detail.title}</h2>
                            <span className="shrink-0 rounded border border-stone-300 px-1.5 text-[11px] text-stone-500 dark:border-stone-700">只读</span>
                        </div>
                        <div className="flex shrink-0 items-center gap-2">
                            <button
                                type="button"
                                className="inline-flex items-center gap-1.5 rounded-xl bg-stone-950 px-3.5 py-2 text-[13px] font-semibold text-stone-50 transition hover:bg-stone-700 dark:bg-stone-100 dark:text-stone-950 dark:hover:bg-stone-300"
                                onClick={() => void forkWork(detail.id)}
                            >
                                <Copy className="size-3.5" />
                                复制项目
                            </button>
                            <button
                                type="button"
                                className="flex size-8 items-center justify-center rounded-lg text-stone-500 transition hover:bg-stone-100 hover:text-stone-950 dark:hover:bg-stone-800 dark:hover:text-stone-100"
                                title="关闭"
                                onClick={closeOverlays}
                            >
                                <X className="size-4" />
                            </button>
                        </div>
                    </header>
                    <ProcessCanvas nodes={processProject.nodes} connections={processProject.connections} groupLabel={`发布于 ${formatWorkDate(detail.createdAt)}`} />
                </section>
            ) : null}
        </main>
    );
}

function formatWorkDate(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return `${date.getMonth() + 1}月${date.getDate()}日`;
}
