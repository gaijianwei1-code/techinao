"use client";

// Next.js 全局错误兜底：客户端渲染异常或发新版本后旧 chunk 404 时，
// 默认会显示英文 "This page couldn't load" 死路页，这里换成可恢复的中文页。
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
    return (
        <html lang="zh-CN">
            <body style={{ margin: 0, minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", background: "#0a0a0a", color: "#fafafa", fontFamily: "system-ui, -apple-system, sans-serif" }}>
                <div style={{ textAlign: "center", display: "grid", gap: 16, padding: 24 }}>
                    <h1 style={{ fontSize: 20, fontWeight: 600, margin: 0 }}>页面加载出错了</h1>
                    <p style={{ color: "#a1a1aa", fontSize: 14, margin: 0, lineHeight: 1.6 }}>
                        如果刚更新过版本，可能是本地缓存过期，刷新一下通常就能恢复。
                    </p>
                    <div style={{ display: "flex", gap: 8, justifyContent: "center" }}>
                        <button
                            onClick={() => window.location.reload()}
                            style={{ padding: "8px 20px", borderRadius: 8, border: "none", background: "#fafafa", color: "#0a0a0a", fontSize: 14, cursor: "pointer" }}
                        >
                            刷新页面
                        </button>
                        <button
                            onClick={reset}
                            style={{ padding: "8px 20px", borderRadius: 8, border: "1px solid #3f3f46", background: "transparent", color: "#fafafa", fontSize: 14, cursor: "pointer" }}
                        >
                            重试
                        </button>
                    </div>
                    <details style={{ maxWidth: 720, textAlign: "left", fontSize: 12, color: "#71717a" }}>
                        <summary style={{ cursor: "pointer" }}>错误详情{error?.digest ? `（${error.digest}）` : ""}</summary>
                        <pre style={{ whiteSpace: "pre-wrap", wordBreak: "break-all", marginTop: 8 }}>{error?.stack || error?.message || String(error)}</pre>
                    </details>
                </div>
            </body>
        </html>
    );
}
