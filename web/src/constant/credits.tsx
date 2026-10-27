import type { ComponentProps } from "react";
import { Zap } from "lucide-react";

export function CreditSymbol({ className, ...props }: ComponentProps<"span">) {
    return (
        <span {...props} className={`inline-flex items-center justify-center ${className || ""}`}>
            <Zap className="size-[1em] fill-current" strokeWidth={2.4} />
        </span>
    );
}

export type ModelCreditCost = {
    model: string;
    credits: number;
};

export function modelCreditCost(modelCosts: ModelCreditCost[] | undefined, model: string) {
    return modelCosts?.find((item) => item.model === model)?.credits || 0;
}

export function requestCreditCost(options: { channelMode: string; modelCosts?: ModelCreditCost[]; model: string; count?: string | number }) {
    if (options.channelMode !== "remote") return 0;
    const count = Math.max(1, Math.floor(Math.abs(Number(options.count)) || 1));
    return modelCreditCost(options.modelCosts, options.model) * count;
}

// 输出尺寸的算力系数，跟后端 service.ImageSizeCreditFactor 保持一致：
// 2K 档（最大边 2048/3136）1.5 倍，4K 档（最大边 3840/6272）2 倍。
export function imageSizeCreditFactor(size: string | undefined) {
    const match = /^(\d+)x(\d+)$/.exec((size || "").trim());
    if (!match) return 1;
    const longest = Math.max(Number(match[1]), Number(match[2]));
    if (longest >= 3800) return 2;
    if (longest >= 2000) return 1.5;
    return 1;
}

// 本次生成的算力消耗：托管账号按次固定 1 点，其它账号按模型单价，再乘张数和尺寸系数。
// 与后端 AICallCredits + ImageSizeCreditFactor 的算法一致；本地直连不计费。
export function generationCreditCost(options: { channelMode: string; managed: boolean; modelCosts?: ModelCreditCost[]; model: string; count?: string | number; size?: string }) {
    if (options.channelMode !== "remote") return 0;
    const count = Math.max(1, Math.floor(Math.abs(Number(options.count)) || 1));
    const base = options.managed ? 1 : modelCreditCost(options.modelCosts, options.model);
    return base * count * imageSizeCreditFactor(options.size);
}

export function formatCredit(value: number) {
    return Number.isInteger(value) ? String(value) : value.toFixed(1);
}
