#!/bin/bash
# 无限画布 · macOS 一键启动（双击运行）
# 对应 Windows 上的「一键启动全部服务.bat」：起应用 → 等就绪 → 起内网穿透 → 复制地址 → 开浏览器。
# 注意：本文件必须保持 LF 换行，改成 CRLF 后 macOS 会因为 shebang 带 \r 而无法双击运行。

set -uo pipefail

# 双击 .command 时工作目录是用户主目录，必须先切到脚本所在目录。
cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1

PORT="${CANVAS_PORT:-3030}"
LOCAL_URL="http://127.0.0.1:${PORT}"
URL_FILE="$(pwd)/.run/tunnel-url.txt"

echo
echo " =================================================="
echo "   无限画布 / 一键启动全部服务（macOS）"
echo "   1. 启动应用（Docker）    2. 等后端就绪"
echo "   3. 内网穿透              4. 复制地址并打开"
echo " =================================================="
echo

pause() {
    echo
    echo " 按回车键关闭本窗口..."
    read -r _answer
}

# ---------- 1/3 启动应用 ----------
echo " [1/3] 启动应用..."
echo "       首次运行需要构建镜像（约 3~10 分钟），期间请勿关闭本窗口。"
echo
if ! bash ./start.sh start; then
    echo
    echo " [x] 应用启动失败，已跳过内网穿透。"
    echo "     排查：bash ./start.sh logs"
    pause
    exit 1
fi

# ---------- 2/3 等后端就绪 ----------
echo
echo " [2/3] 等待后端就绪..."
READY=0
for i in $(seq 1 60); do
    if [ "$(curl -s -m 4 "${LOCAL_URL}/api/health" 2>/dev/null || true)" = "ok" ]; then
        echo "       后端已就绪（第 ${i} 次探测）"
        READY=1
        break
    fi
    echo "       等待中（${i}/60）..."
    sleep 3
done
if [ "$READY" != "1" ]; then
    echo
    echo " [x] 后端 180 秒内没就绪，已跳过内网穿透（否则朋友打开会看到 502）。"
    echo "     排查：bash ./start.sh logs"
    pause
    exit 1
fi

# ---------- 3/3 内网穿透 ----------
echo
echo " [3/3] 启动内网穿透（cloudflared 临时域名）..."
if ! bash ./scripts/tunnel/tunnel.sh cloudflared; then
    echo
    echo " [x] 应用已就绪，但没拿到对外地址。"
    echo "     日志：$(pwd)/logs/tunnel-cloudflared.log"
    echo "     常见原因：没装 cloudflared、连不上 Cloudflare 边缘。"
    echo "     本机访问不受影响：${LOCAL_URL}"
    pause
    exit 1
fi

if [ ! -f "$URL_FILE" ]; then
    echo " [x] 没找到地址文件：${URL_FILE}"
    pause
    exit 1
fi
PUBLIC_URL="$(cat "$URL_FILE")"
if [ -z "$PUBLIC_URL" ]; then
    echo " [x] 地址文件是空的：${URL_FILE}"
    pause
    exit 1
fi

printf '%s' "$PUBLIC_URL" | pbcopy

echo
echo " --------------------------------------------------"
echo "   本机地址：${LOCAL_URL}"
echo "   对外地址：${PUBLIC_URL}"
echo " --------------------------------------------------"
echo "   [i] 对外地址已复制到剪贴板，直接发给朋友即可。"
echo "   [i] 临时域名：重启或重跑本脚本都会换地址，记得重新发一次。"
echo

echo " 正在用本机验证公网地址（新域名本机 DNS 可能还没同步）..."
if [ "$(curl -s -m 25 "${PUBLIC_URL}/api/health" 2>/dev/null || true)" = "ok" ]; then
    echo "   [i] 公网自测通过，朋友那边可以直接打开。"
    open "$PUBLIC_URL"
    echo "   [i] 已在本机浏览器打开对外地址。"
else
    echo "   [注意] 本机打不开对外地址，这多半是本机 DNS 把新域名缓存成了不存在，属于正常现象。"
    echo "          朋友那边能打开；本机想打开就把 DNS 换成 119.29.29.29。"
    open "$LOCAL_URL"
    echo "   [i] 已在本机浏览器打开 ${LOCAL_URL}。"
fi

echo
echo " --------------------------------------------------"
echo "   停止全部：bash ./start.sh stop"
echo "   只关隧道：bash ./scripts/tunnel/tunnel.sh stop"
echo " --------------------------------------------------"
echo "   [i] 关闭本窗口不会停止任何服务。"
echo "   [i] 首次给朋友用之前，记得去「后台 - 用户管理」给他建一个托管账号并设置使用次数，"
echo "       建议同时在「后台 - 系统设置」关掉「是否允许用户注册」。"
pause
