#!/usr/bin/env bash
# 无限画布 · 内网穿透（cloudflared 临时 https 域名）
#
#   bash scripts/tunnel/tunnel.sh cloudflared   # 起隧道，打印 https://xxx.trycloudflare.com
#   bash scripts/tunnel/tunnel.sh url           # 只打印当前对外地址
#   bash scripts/tunnel/tunnel.sh status        # 隧道状态
#   bash scripts/tunnel/tunnel.sh stop          # 关掉隧道（应用本身不受影响）
#
# 环境变量：
#   CANVAS_PORT         穿透目标端口，默认 3030（Docker 版映射端口；Windows 安装版是 3000）
#   CANVAS_CF_PROTOCOL  cloudflared 传输协议，默认 http2。**不要改回 quic**：
#                       在 WSL2 上 UDP 缓冲只有 208 KiB（cloudflared 想要 ~7 MiB），quic 吞吐会被
#                       压到 ~100 KB/s，首页能开但主 JS 包下不来 → 朋友看到白屏；实测 http2 快十倍以上。
#                       macOS / 原生 Linux 没这个限制，但 http2 更稳，保持默认即可。
#   CANVAS_CF_BIN       cloudflared 路径。默认先找 PATH（Homebrew 装在 /opt/homebrew/bin
#                       或 /usr/local/bin），再回退 ~/.local/bin/cloudflared（Linux 手动装的位置）。
#
# 免费临时域名：每次启动地址都会变，重启/重跑后要把新地址重新发给朋友。要固定地址只能自建 frp。
set -uo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_DIR="$APP_DIR/.run"
LOG_DIR="$APP_DIR/logs"
LOG_FILE="$LOG_DIR/tunnel-cloudflared.log"
PID_FILE="$RUN_DIR/tunnel-cloudflared.pid"
URL_FILE="$RUN_DIR/tunnel-url.txt"

PORT="${CANVAS_PORT:-3030}"
PROTOCOL="${CANVAS_CF_PROTOCOL:-http2}"
CF_BIN="${CANVAS_CF_BIN:-$HOME/.local/bin/cloudflared}"
# 用 127.0.0.1 而不是 localhost：macOS 上 localhost 常优先解析到 ::1，Docker Desktop
# 的 IPv6 loopback 转发不如 IPv4 稳，会让隧道拿到 502。
LOCAL_URL="http://127.0.0.1:$PORT"
# 认自己的隧道进程时**必须带上端口**：同一台机器上可能还跑着别的项目的 cloudflared，
# 只匹配 "cloudflared ... --url http://127.0.0.1:" 会把人家的隧道一起 kill 掉。
# 同时兼容老版本写下的 localhost 地址，免得旧隧道认不出来。
ORPHAN_PATTERN="cloudflared tunnel --no-autoupdate.*--url http://(127\.0\.0\.1|localhost):${PORT}( |\$)"

info() { printf '[信息] %s\n' "$1"; }
ok() { printf '[成功] %s\n' "$1"; }
warn() { printf '[注意] %s\n' "$1"; }
err() { printf '[错误] %s\n' "$1"; }

cloudflared_bin() {
    # 显式指定优先，其次 PATH（Homebrew），最后才是 $CF_BIN 默认路径。
    if [ -n "${CANVAS_CF_BIN:-}" ] && [ -x "$CANVAS_CF_BIN" ]; then
        printf '%s' "$CANVAS_CF_BIN"
        return 0
    fi
    if command -v cloudflared >/dev/null 2>&1; then
        command -v cloudflared
        return 0
    fi
    if [ -x "$CF_BIN" ]; then
        printf '%s' "$CF_BIN"
        return 0
    fi
    return 1
}

# pid 文件失效时按命令行特征兜底认，避免出现「隧道在跑、status 说没跑」
alive_pid() {
    local pid=""
    if [ -f "$PID_FILE" ]; then
        pid="$(cat "$PID_FILE" 2>/dev/null)"
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            printf '%s' "$pid"
            return 0
        fi
    fi
    pgrep -f "$ORPHAN_PATTERN" 2>/dev/null | head -n1
}

kill_tunnel() {
    local pid i
    pid="$(alive_pid)"
    if [ -n "$pid" ]; then
        kill "$pid" 2>/dev/null
        for i in 1 2 3 4 5 6 7 8 9 10; do
            kill -0 "$pid" 2>/dev/null || break
            sleep 0.3
        done
        kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null
    fi
    pkill -f "$ORPHAN_PATTERN" 2>/dev/null
    rm -f "$PID_FILE" "$URL_FILE"
}

local_health() {
    curl -s -m 5 "$LOCAL_URL/api/health" 2>/dev/null | tr -d '\r\n'
}

preflight() {
    [ "$(local_health)" = "ok" ] && return 0
    warn "本机 $LOCAL_URL/api/health 没返回 ok，应用可能还没启动。"
    warn "先在 Windows 双击「无限画布-一键启动」（或 WSL 里 ./start.sh start），隧道会一直等它起来。"
}

# 公网自测。本机 DNS 常把刚建的新域名缓存成 NXDOMAIN，所以直连失败时改问公共 DNS 再 --resolve 验一次。
public_health() {
    local host="${1#https://}" ip code
    code="$(curl -s -m 20 -o /dev/null -w '%{http_code}' "$1/api/health" 2>/dev/null)"
    [ "$code" = "200" ] && { printf 'direct 200'; return 0; }
    if command -v dig >/dev/null 2>&1; then
        ip="$(dig +short +time=3 +tries=1 @119.29.29.29 "$host" A 2>/dev/null | grep -E '^[0-9.]+$' | head -n1)"
        if [ -n "$ip" ]; then
            code="$(curl -s -m 20 -o /dev/null -w '%{http_code}' --resolve "$host:443:$ip" "$1/api/health" 2>/dev/null)"
            printf 'public-dns %s' "${code:-000}"
            return 0
        fi
    fi
    printf 'fail %s' "${code:-000}"
}

# 新域名刚建好时本机和公共 DNS 都可能查不到，隔几秒重试几次再下结论
wait_public() {
    local url="$1" attempt result="fail 000"
    for attempt in 1 2 3 4 5 6; do
        result="$(public_health "$url")"
        case "$result" in
        "direct 200" | "public-dns 200")
            printf '%s' "$result"
            return 0
            ;;
        esac
        [ "$attempt" -lt 6 ] && sleep 10
    done
    printf '%s' "$result"
    return 1
}

start_tunnel() {
    local bin url i
    if ! bin="$(cloudflared_bin)"; then
        err "没找到 cloudflared，装一个再试："
        if [ "$(uname -s)" = "Darwin" ]; then
            echo "  macOS：brew install cloudflared"
            echo "         或从 https://github.com/cloudflare/cloudflared/releases/latest 手动下载"
            echo "         （Apple Silicon 选 cloudflared-darwin-arm64.tgz，Intel 选 -amd64.tgz，解压后把二进制放进 PATH）"
        else
            echo "  Linux：mkdir -p ~/.local/bin && curl -L -o ~/.local/bin/cloudflared https://ghfast.top/https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 && chmod +x ~/.local/bin/cloudflared"
        fi
        return 1
    fi
    mkdir -p "$RUN_DIR" "$LOG_DIR"
    preflight
    if [ -n "$(alive_pid)" ]; then
        info "隧道已在运行，先重启它拿新地址…"
        kill_tunnel
    fi

    : >"$LOG_FILE"
    # setsid 让隧道脱离当前会话，wsl.exe 退出后照样跑
    if command -v setsid >/dev/null 2>&1; then
        setsid "$bin" tunnel --no-autoupdate --protocol "$PROTOCOL" --url "$LOCAL_URL" >>"$LOG_FILE" 2>&1 &
    else
        nohup "$bin" tunnel --no-autoupdate --protocol "$PROTOCOL" --url "$LOCAL_URL" >>"$LOG_FILE" 2>&1 &
    fi
    echo $! >"$PID_FILE"
    info "cloudflared 已启动（协议 $PROTOCOL，目标 $LOCAL_URL），等它分配公网地址…"

    url=""
    for i in $(seq 1 60); do
        url="$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG_FILE" 2>/dev/null | head -n1)"
        [ -n "$url" ] && break
        sleep 1
    done
    if [ -z "$url" ]; then
        err "60 秒内没拿到公网地址，看日志：$LOG_FILE"
        tail -n 5 "$LOG_FILE" 2>/dev/null
        return 1
    fi
    printf '%s\n' "$url" >"$URL_FILE"
    ok "对外地址：$url"

    info "等 Cloudflare 把新域名挂出去（刚建好的域名本机/公共 DNS 都可能查不到，最多等 50 秒）…"
    case "$(wait_public "$url")" in
    "direct 200")
        ok "公网自测通过（$url/api/health -> 200），可以直接发给朋友。"
        ;;
    "public-dns 200")
        warn "公共 DNS 已能解析并返回 200，但本机 DNS 还没同步到这个新域名：你本机可能打不开、朋友能打开。"
        warn "本机想打开就把 DNS 换成 119.29.29.29（114.114.114.114、223.5.5.5 实测对新域名会无应答）。"
        ;;
    *)
        warn "公网自测没通过。"
        warn "一直 502/530 = 应用没在跑（$LOCAL_URL/api/health 要是 ok）或隧道抖动，重跑一次本脚本即可。"
        warn "本机打不开、朋友却能打开 = 本机 DNS 缓存了新域名，把本机 DNS 换成 119.29.29.29 再试。"
        ;;
    esac
    echo
    echo "  发给朋友的地址：$url"
    echo "  关闭隧道：双击「无限画布-关闭外网访问」，或 bash scripts/tunnel/tunnel.sh stop"
    echo "  临时域名重启会变，重跑本脚本会拿到新地址。"
    return 0
}

show_url() {
    if [ -f "$URL_FILE" ]; then
        cat "$URL_FILE"
        return 0
    fi
    info "还没有对外地址，先跑：bash scripts/tunnel/tunnel.sh cloudflared"
    return 1
}

show_status() {
    local pid url proto
    pid="$(alive_pid)"
    url="$(cat "$URL_FILE" 2>/dev/null || true)"
    proto="$(grep -m1 -oE 'Initial protocol [a-z0-9]+' "$LOG_FILE" 2>/dev/null | awk '{print $3}')"
    if [ -n "$pid" ]; then
        ok "隧道运行中：pid=$pid 目标=$LOCAL_URL 协议=${proto:-未知}"
    else
        info "隧道未运行。"
    fi
    if [ -n "$url" ]; then
        echo "  对外地址：$url"
    else
        echo "  对外地址：还没有"
    fi
    echo "  应用本机健康检查：$(local_health)"
    echo "  日志：$LOG_FILE"
}

stop_tunnel() {
    if [ -z "$(alive_pid)" ]; then
        info "隧道本来就没在运行。"
        rm -f "$PID_FILE" "$URL_FILE"
        return 0
    fi
    kill_tunnel
    ok "隧道已关闭，对外地址失效；应用本身没动，本机 $LOCAL_URL 照常可用。"
}

case "${1:-cloudflared}" in
cloudflared) start_tunnel ;;
url) show_url ;;
status) show_status ;;
stop) stop_tunnel ;;
help | -h | --help)
    sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'
    ;;
*)
    err "未知参数：$1"
    echo "用法：bash scripts/tunnel/tunnel.sh {cloudflared|url|status|stop}"
    exit 1
    ;;
esac
