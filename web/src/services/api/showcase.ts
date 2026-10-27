import type { CanvasProject } from "@/app/(user)/canvas/stores/use-canvas-store";
import { apiGet, apiPost, type ApiParams } from "@/services/api/request";
import { useUserStore } from "@/stores/use-user-store";

export type ShowcaseWorkSummary = {
    id: string;
    userId: string;
    title: string;
    category: string;
    cover: string;
    authorName: string;
    authorAvatar: string;
    likes: number;
    liked: boolean;
    createdAt: string;
};

export type ShowcaseWorkDetail = ShowcaseWorkSummary & {
    description: string;
    preview: string;
    projectId: string;
    projectData: string;
    status: string;
    updatedAt: string;
};

export type ShowcaseWorkList = {
    items: ShowcaseWorkSummary[];
    total: number;
    categories: string[];
};

export type PublishShowcasePayload = {
    projectId: string;
    title: string;
    description: string;
    category: string;
    cover: string;
    preview: string;
    data: CanvasProject;
};

// 与后端 model.ShowcaseCategories 保持一致。
export const SHOWCASE_CATEGORIES = ["精选画布", "专业影视", "短剧漫剧", "动漫游戏", "教育生活"];

function token() {
    return useUserStore.getState().token || undefined;
}

export function listShowcaseWorks(params: ApiParams) {
    // 查询接口是公开路由（/api/showcase，OptionalAuth），不带 /v1 前缀。
    return apiGet<ShowcaseWorkList>("/api/showcase", params, token());
}

export function getShowcaseWork(id: string) {
    return apiGet<ShowcaseWorkDetail>(`/api/showcase/${encodeURIComponent(id)}`, undefined, token());
}

export function publishShowcaseWork(payload: PublishShowcasePayload) {
    return apiPost<ShowcaseWorkSummary>("/api/v1/showcase", payload, token());
}

export function likeShowcaseWork(id: string) {
    return apiPost<{ liked: boolean; likes: number }>(`/api/v1/showcase/${encodeURIComponent(id)}/like`, {}, token());
}

export function forkShowcaseWork(id: string) {
    return apiPost<{ data: CanvasProject }>(`/api/v1/showcase/${encodeURIComponent(id)}/fork`, {}, token());
}

// pickShowcaseMedia 从画布项目节点中提取预览视频与封面（视频优先）。
export function pickShowcaseMedia(project: Pick<CanvasProject, "nodes">) {
    let preview = "";
    let cover = "";
    for (const node of project.nodes) {
        const content = node.metadata?.content || "";
        if (!content || content.startsWith("data:")) continue;
        if (node.type === "video" && !preview) preview = content;
        if (node.type === "image" && !cover) cover = content;
    }
    if (!cover) cover = preview;
    return { preview, cover };
}
