import { apiGet, compactApiParams } from "@/services/api/request";

export type Prompt = {
    id: string;
    title: string;
    coverUrl: string;
    prompt: string;
    tags: string[];
    category: string;
    githubUrl: string;
    preview: string;
    createdAt: string;
    updatedAt: string;
    description?: string;
    referenceImageUrls?: string[];
};

export const ALL_PROMPTS_OPTION = "全部";

export type PromptCategoryOption = {
    category: string;
    name: string;
};

export type PromptListResponse = {
    items: Prompt[];
    tags: string[];
    categories: PromptCategoryOption[];
    total: number;
};

export async function fetchPrompts({ keyword = "", tag = [], category = ALL_PROMPTS_OPTION, page, pageSize }: { keyword?: string; tag?: string[]; category?: string; page?: number; pageSize?: number } = {}) {
    return apiGet<PromptListResponse>(
        "/api/prompts",
        compactApiParams({
            ...(keyword ? { keyword } : {}),
            ...(tag.length ? { tag } : {}),
            ...(category !== ALL_PROMPTS_OPTION ? { category } : {}),
            ...(page ? { page } : {}),
            ...(pageSize ? { pageSize } : {}),
        }),
    );
}

// 画布提示词面板用：sourceId 兼容参考仓库的提示词源概念，这里统一走后端提示词库（1588 条本地库）
export async function fetchSourcePrompts(sourceId: string): Promise<Prompt[]> {
    const response = await fetchPrompts({ category: sourceId, pageSize: 500 }).catch(() => null);
    if (response?.items?.length) return response.items;
    const fallback = await fetchPrompts({ pageSize: 500 });
    return fallback.items;
}

export function formatPromptDate(value: string, _locale?: string) {
    void _locale;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" }).format(date);
}
