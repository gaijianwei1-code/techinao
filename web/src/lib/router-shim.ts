"use client";

// react-router-dom 兼容 shim：画布页面从参考仓库移植，用 Next.js 路由实现同名 API
import { useCallback } from "react";
import { useParams as useNextParams, useRouter, useSearchParams as useNextSearchParams } from "next/navigation";

export type NavigateOptions = { replace?: boolean; state?: unknown };

export function useNavigate() {
    const router = useRouter();
    // react-router 的 navigate 是稳定引用，页面会把它放进 useEffect 依赖，必须保持稳定
    return useCallback(
        (href: string, options?: NavigateOptions) => {
            if (options?.replace) router.replace(href);
            else router.push(href);
        },
        [router],
    );
}

export function useParams<T = Record<string, string | undefined>>(): T {
    return useNextParams() as T;
}

export function useSearchParams(): [URLSearchParams, never] {
    const params = useNextSearchParams();
    return [params as unknown as URLSearchParams, undefined as never];
}
