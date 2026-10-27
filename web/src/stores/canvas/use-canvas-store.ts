// 画布 UI 统一使用 app/(user)/canvas/stores 的增强版 store（含云端同步、agent 面板、回收站）
export {
    DEFAULT_CANVAS_AGENT_PANEL,
    DEFAULT_CANVAS_SIDE_PANEL,
    mergeCanvasProjects,
    useCanvasStore,
    type CanvasDeletedProject,
    type CanvasProject,
    type CanvasSidePanelState,
} from "@/app/(user)/canvas/stores/use-canvas-store";
