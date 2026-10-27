// 画布重度依赖浏览器本地存储与插件运行时，禁止构建期预渲染
export const dynamic = "force-dynamic";

import CanvasIndex from "@/views/canvas/index";

export default function CanvasPage() {
    return <CanvasIndex />;
}
