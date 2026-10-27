import Link from "next/link";
import { ArrowUpRight, FileText, FolderOpen, ImagePlus, Images, LayoutGrid, Sparkles, Workflow, Video } from "lucide-react";

type Entry = {
    href: string;
    title: string;
    desc: string;
    icon: typeof ImagePlus;
};

const entries: Entry[] = [
    { href: "/canvas", title: "无限画布", desc: "节点式画布：生成、编辑、连线编排，多项目管理", icon: LayoutGrid },
    { href: "/image", title: "生图工作台", desc: "写提示词、带参考图，直接出图", icon: ImagePlus },
    { href: "/video", title: "视频创作台", desc: "文生视频 / 图生视频，带首尾帧与运镜设置", icon: Video },
    { href: "/showcase", title: "创作精选", desc: "浏览大家的画布作品，看成片、看创作过程", icon: Sparkles },
    { href: "/prompts", title: "提示词中心", desc: "浏览、收藏、复用提示词", icon: FileText },
    { href: "/assets", title: "我的素材", desc: "图片 / 视频 / 文本素材统一存放", icon: Images },
    { href: "/asset-library", title: "素材库", desc: "跨项目浏览与整理素材", icon: FolderOpen },
    { href: "/workflows", title: "创意工作流", desc: "把重复的创作流程固化下来", icon: Workflow },
];

export default function IndexPage() {
    return (
        <div className="h-full overflow-y-auto">
            <div className="mx-auto max-w-5xl px-6 py-16">
                <span className="inline-flex items-center rounded-full border border-stone-200 px-3 py-1 text-xs text-stone-500 dark:border-stone-800 dark:text-stone-400">
                    本地部署 · 自用
                </span>
                <h1 className="mt-5 text-4xl font-semibold tracking-tight text-stone-950 dark:text-stone-100">我自己的 AI 工作台</h1>
                <p className="mt-4 max-w-2xl text-base leading-7 text-stone-500 dark:text-stone-400">
                    生图、提示词、素材、工作流收在一个界面里，只给自己用。
                </p>

                <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {entries.map(({ href, title, desc, icon: Icon }) => (
                        <Link
                            key={href}
                            href={href}
                            className="group relative flex flex-col gap-3 rounded-2xl border border-stone-200 bg-white/60 p-5 transition hover:-translate-y-0.5 hover:border-stone-300 hover:shadow-sm dark:border-stone-800 dark:bg-stone-900/40 dark:hover:border-stone-700"
                        >
                            <span className="inline-flex size-9 items-center justify-center rounded-xl bg-stone-100 text-stone-700 dark:bg-stone-800 dark:text-stone-200">
                                <Icon className="size-4" />
                            </span>
                            <div>
                                <p className="text-sm font-medium text-stone-950 dark:text-stone-100">{title}</p>
                                <p className="mt-1 text-xs leading-5 text-stone-500 dark:text-stone-400">{desc}</p>
                            </div>
                            <ArrowUpRight className="absolute right-4 top-4 size-4 text-stone-300 transition group-hover:text-stone-500 dark:text-stone-700 dark:group-hover:text-stone-400" />
                        </Link>
                    ))}
                </div>

                <p className="mt-14 text-xs text-stone-400 dark:text-stone-600">本地部署 · 端口 3030</p>
            </div>
        </div>
    );
}
