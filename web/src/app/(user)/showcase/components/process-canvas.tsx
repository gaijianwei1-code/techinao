"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent as ReactMouseEvent, type PointerEvent as ReactPointerEvent } from "react";
import { AudioLines, Boxes, Film, Image as ImageIcon, Map as MapIcon, Minus, Plus, Spline, Type } from "lucide-react";

import { canvasThemes } from "@/lib/canvas-theme";
import { useThemeStore } from "@/stores/use-theme-store";
import { CanvasNodeType, type CanvasConnection, type CanvasNodeData } from "@/app/(user)/canvas/types";

type Transform = { x: number; y: number; k: number };

const MIN_K = 0.4;
const MAX_K = 2;
const MINIMAP_W = 176;
const MINIMAP_H = 108;

const NODE_ICONS: Record<string, typeof Film> = {
    [CanvasNodeType.Video]: Film,
    [CanvasNodeType.Image]: ImageIcon,
    [CanvasNodeType.Panorama]: ImageIcon,
    [CanvasNodeType.Text]: Type,
    [CanvasNodeType.Audio]: AudioLines,
    [CanvasNodeType.Config]: Boxes,
    [CanvasNodeType.Group]: Boxes,
    [CanvasNodeType.Director]: Film,
};

function worldBounds(nodes: CanvasNodeData[]) {
    if (!nodes.length) return { minX: 0, minY: 0, maxX: 1000, maxY: 600 };
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    for (const node of nodes) {
        minX = Math.min(minX, node.position.x);
        minY = Math.min(minY, node.position.y);
        maxX = Math.max(maxX, node.position.x + node.width);
        maxY = Math.max(maxY, node.position.y + node.height + 60);
    }
    return { minX, minY, maxX, maxY };
}

function isMediaVideo(url: string) {
    return /\.(mp4|webm|mov|m4v)(\?|#|$)/i.test(url);
}

// ProcessCanvas 只读画布回放：节点 + 连线 + 平移缩放 + 小地图。
export function ProcessCanvas({
    nodes,
    connections,
    groupLabel,
}: {
    nodes: CanvasNodeData[];
    connections: CanvasConnection[];
    groupLabel: string;
}) {
    const theme = useThemeStore((state) => state.theme);
    const palette = canvasThemes[theme];
    const viewportRef = useRef<HTMLDivElement>(null);
    const minimapRef = useRef<HTMLCanvasElement>(null);
    const panRef = useRef<{ sx: number; sy: number; tx: number; ty: number } | null>(null);
    const [transform, setTransform] = useState<Transform>({ x: 0, y: 0, k: 1 });
    const [showEdges, setShowEdges] = useState(true);
    const [showMinimap, setShowMinimap] = useState(false);

    const bounds = useMemo(() => worldBounds(nodes), [nodes]);
    const resolution = useMemo(() => nodes.find((node) => node.type === CanvasNodeType.Video)?.metadata?.size || "", [nodes]);

    const applyOrClamp = useCallback((next: Transform) => {
        setTransform({ x: next.x, y: next.y, k: Math.max(MIN_K, Math.min(MAX_K, next.k)) });
    }, []);

    const fitView = useCallback(() => {
        const viewport = viewportRef.current;
        if (!viewport) return;
        const width = bounds.maxX - bounds.minX;
        const height = bounds.maxY - bounds.minY;
        const pad = 80;
        const k = Math.max(MIN_K, Math.min(MAX_K, Math.min((viewport.clientWidth - pad * 2) / width, (viewport.clientHeight - pad * 2) / height, 1)));
        applyOrClamp({
            x: (viewport.clientWidth - width * k) / 2 - bounds.minX * k,
            y: (viewport.clientHeight - height * k) / 2 - bounds.minY * k,
            k,
        });
    }, [applyOrClamp, bounds]);

    useEffect(() => {
        fitView();
        const onResize = () => fitView();
        window.addEventListener("resize", onResize);
        return () => window.removeEventListener("resize", onResize);
    }, [fitView]);

    // 滚轮缩放（以光标为中心）。
    useEffect(() => {
        const viewport = viewportRef.current;
        if (!viewport) return;
        const onWheel = (event: WheelEvent) => {
            event.preventDefault();
            const rect = viewport.getBoundingClientRect();
            setTransform((current) => {
                const k = Math.max(MIN_K, Math.min(MAX_K, current.k * (event.deltaY < 0 ? 1.1 : 1 / 1.1)));
                const cx = event.clientX - rect.left;
                const cy = event.clientY - rect.top;
                return { k, x: cx - ((cx - current.x) * k) / current.k, y: cy - ((cy - current.y) * k) / current.k };
            });
        };
        viewport.addEventListener("wheel", onWheel, { passive: false });
        return () => viewport.removeEventListener("wheel", onWheel);
    }, []);

    const onPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
        if (event.button !== 0) return;
        const target = event.target as HTMLElement;
        if (target.closest("video, audio, button")) return;
        panRef.current = { sx: event.clientX, sy: event.clientY, tx: transform.x, ty: transform.y };
        event.currentTarget.setPointerCapture(event.pointerId);
    };
    const onPointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
        const pan = panRef.current;
        if (!pan) return;
        setTransform((current) => ({ ...current, x: pan.tx + event.clientX - pan.sx, y: pan.ty + event.clientY - pan.sy }));
    };
    const endPan = () => {
        panRef.current = null;
    };

    const edges = useMemo(() => {
        const nodeMap = new Map(nodes.map((node) => [node.id, node]));
        const width = bounds.maxX - bounds.minX + 200;
        const height = bounds.maxY - bounds.minY + 200;
        const paths = connections.map((connection) => {
            const from = nodeMap.get(connection.fromNodeId);
            const to = nodeMap.get(connection.toNodeId);
            if (!from || !to) return null;
            const x1 = from.position.x + from.width;
            const y1 = from.position.y + from.height / 2;
            const x2 = to.position.x;
            const y2 = to.position.y + to.height / 2;
            const dx = Math.max(48, Math.abs(x2 - x1) * 0.5);
            return `M${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
        });
        return { width, height, paths: paths.filter(Boolean) as string[] };
    }, [connections, nodes, bounds]);

    // 小地图：节点矩形 + 当前视口框。
    useEffect(() => {
        if (!showMinimap) return;
        const canvas = minimapRef.current;
        const ctx = canvas?.getContext("2d");
        if (!canvas || !ctx) return;
        ctx.clearRect(0, 0, MINIMAP_W, MINIMAP_H);
        ctx.fillStyle = palette.canvas.background;
        ctx.fillRect(0, 0, MINIMAP_W, MINIMAP_H);
        const width = bounds.maxX - bounds.minX;
        const height = bounds.maxY - bounds.minY;
        const scale = Math.min((MINIMAP_W - 16) / width, (MINIMAP_H - 16) / height);
        const ox = (MINIMAP_W - width * scale) / 2 - bounds.minX * scale;
        const oy = (MINIMAP_H - height * scale) / 2 - bounds.minY * scale;
        for (const node of nodes) {
            ctx.fillStyle = node.type === CanvasNodeType.Video ? "rgba(245,245,244,.5)" : "rgba(245,245,244,.22)";
            ctx.fillRect(ox + node.position.x * scale, oy + node.position.y * scale, Math.max(3, node.width * scale), Math.max(2, node.height * scale));
        }
        const viewport = viewportRef.current;
        if (viewport) {
            ctx.strokeStyle = palette.node.activeStroke;
            ctx.lineWidth = 1.2;
            ctx.strokeRect(
                ox - (transform.x / transform.k) * scale,
                oy - (transform.y / transform.k) * scale,
                (viewport.clientWidth / transform.k) * scale,
                (viewport.clientHeight / transform.k) * scale,
            );
        }
    }, [showMinimap, transform, nodes, bounds, palette]);

    const jumpFromMinimap = (event: ReactMouseEvent<HTMLDivElement>) => {
        const rect = event.currentTarget.getBoundingClientRect();
        const width = bounds.maxX - bounds.minX;
        const height = bounds.maxY - bounds.minY;
        const scale = Math.min((MINIMAP_W - 16) / width, (MINIMAP_H - 16) / height);
        const ox = (MINIMAP_W - width * scale) / 2 - bounds.minX * scale;
        const oy = (MINIMAP_H - height * scale) / 2 - bounds.minY * scale;
        const wx = (event.clientX - rect.left - ox) / scale;
        const wy = (event.clientY - rect.top - oy) / scale;
        const viewport = viewportRef.current;
        if (!viewport) return;
        applyOrClamp({ ...transform, x: viewport.clientWidth / 2 - wx * transform.k, y: viewport.clientHeight / 2 - wy * transform.k });
    };

    const renderNodeBody = (node: CanvasNodeData) => {
        const content = node.metadata?.content || "";
        if ((node.type === CanvasNodeType.Video || node.type === CanvasNodeType.Image || node.type === CanvasNodeType.Panorama) && content) {
            if (node.type === CanvasNodeType.Video || isMediaVideo(content)) {
                return <video src={content} controls preload="metadata" playsInline className="h-full w-full bg-black object-cover" />;
            }
            return <img src={content} alt={node.title} loading="lazy" className="h-full w-full object-cover" />;
        }
        if (node.type === CanvasNodeType.Audio && content) {
            return <audio src={content} controls className="w-full px-2" />;
        }
        return (
            <div className="h-full overflow-hidden p-3 text-xs leading-relaxed whitespace-pre-wrap" style={{ color: palette.node.text }}>
                {content || node.title}
            </div>
        );
    };

    return (
        <div
            ref={viewportRef}
            className="relative min-h-0 flex-1 cursor-grab overflow-hidden active:cursor-grabbing"
            style={{
                backgroundColor: palette.canvas.background,
                backgroundImage: `radial-gradient(${palette.canvas.dot} 1px, transparent 1px)`,
                backgroundSize: `${24 * transform.k}px ${24 * transform.k}px`,
                backgroundPosition: `${transform.x}px ${transform.y}px`,
            }}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={endPan}
            onPointerCancel={endPan}
        >
            <div
                className="absolute top-0 left-0 origin-top-left will-change-transform"
                style={{ transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.k})` }}
            >
                <svg
                    className="pointer-events-none absolute overflow-visible"
                    style={{ left: bounds.minX - 100, top: bounds.minY - 100 }}
                    width={edges.width}
                    height={edges.height}
                    viewBox={`0 0 ${edges.width} ${edges.height}`}
                >
                    <g transform={`translate(${100 - bounds.minX}, ${100 - bounds.minY})`}>
                        {showEdges &&
                            edges.paths.map((path, index) => (
                                <path key={index} d={path} fill="none" stroke={palette.canvas.line} strokeWidth={1.5} />
                            ))}
                    </g>
                </svg>

                <div
                    className="pointer-events-none absolute flex items-center gap-1.5 text-[13px]"
                    style={{ left: bounds.minX, top: bounds.minY - 36, color: palette.node.label }}
                >
                    <Film className="size-3" />
                    <span>
                        {groupLabel}({nodes.length})
                    </span>
                    {resolution ? <span className="ml-2 font-mono text-xs" style={{ color: palette.node.faint }}>{resolution}</span> : null}
                </div>

                {nodes.map((node) => {
                    const Icon = NODE_ICONS[node.type] || Boxes;
                    return (
                        <div
                            key={node.id}
                            className="absolute overflow-hidden rounded-[10px] border"
                            style={{
                                left: node.position.x,
                                top: node.position.y,
                                width: node.width,
                                height: node.height,
                                background: palette.node.panel,
                                borderColor: node.type === CanvasNodeType.Config ? palette.node.activeStroke : palette.node.stroke,
                                boxShadow: "0 8px 24px rgba(0,0,0,.35)",
                            }}
                        >
                            <div
                                className="flex h-7 items-center gap-1.5 border-b px-2.5 text-[11px]"
                                style={{ borderColor: palette.canvas.line, color: node.type === CanvasNodeType.Config ? palette.node.activeStroke : palette.node.label }}
                            >
                                <Icon className="size-3 shrink-0" />
                                <span className="truncate">{node.title}</span>
                            </div>
                            <div className="h-[calc(100%-28px)]">{renderNodeBody(node)}</div>
                        </div>
                    );
                })}
            </div>

            {/* 左下扁平工具栏 */}
            <div className="absolute bottom-4 left-4 flex items-center gap-1">
                <button
                    type="button"
                    className={`flex size-8 items-center justify-center rounded-lg transition ${showMinimap ? "bg-stone-200/20" : "hover:bg-stone-200/10"}`}
                    style={{ color: palette.node.label }}
                    title="切换小地图"
                    onClick={() => setShowMinimap((value) => !value)}
                >
                    <MapIcon className="size-4" />
                </button>
                <button
                    type="button"
                    className={`flex size-8 items-center justify-center rounded-lg transition ${showEdges ? "" : "bg-stone-200/20"}`}
                    style={{ color: palette.node.label }}
                    title={showEdges ? "隐藏节点连线" : "显示节点连线"}
                    onClick={() => setShowEdges((value) => !value)}
                >
                    <Spline className="size-4" />
                </button>
                <button
                    type="button"
                    className="flex size-8 items-center justify-center rounded-lg hover:bg-stone-200/10"
                    style={{ color: palette.node.label }}
                    title="缩小"
                    onClick={() => applyOrClamp({ ...transform, k: transform.k / 1.2 })}
                >
                    <Minus className="size-4" />
                </button>
                <button
                    type="button"
                    className="flex size-8 items-center justify-center rounded-lg hover:bg-stone-200/10"
                    style={{ color: palette.node.label }}
                    title="放大"
                    onClick={() => applyOrClamp({ ...transform, k: transform.k * 1.2 })}
                >
                    <Plus className="size-4" />
                </button>
                <span className="min-w-11 text-center font-mono text-xs" style={{ color: palette.node.label }}>
                    {Math.round(transform.k * 100)}%
                </span>
            </div>

            {showMinimap ? (
                <div
                    className="absolute bottom-14 left-4 cursor-pointer overflow-hidden rounded-lg border shadow-lg"
                    style={{ borderColor: palette.node.stroke }}
                    onClick={jumpFromMinimap}
                >
                    <canvas ref={minimapRef} width={MINIMAP_W} height={MINIMAP_H} />
                </div>
            ) : null}
        </div>
    );
}
