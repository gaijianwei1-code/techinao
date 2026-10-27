#!/usr/bin/env bash
# 无限画布 (infinite-canvas) 一键启动脚本
# 用法: ./start.sh [start|stop|restart|rebuild|logs|status|update|help]
#       不带参数 = start

set -uo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="docker-compose.local.yml"   # 纯源码构建（docker-compose.yml 带 pull_policy，会先拉 ghcr）
IMAGE="infinite-canvas:local"
CONTAINER="infinite-canvas-app-1"
HOST_PORT="3030"                          # 宿主机端口（3000 被 promptpilot 占用）
INNER_PORT="3000"

cd "$APP_DIR" || exit 1

# ---------- 颜色 ----------
if [ -t 1 ]; then
  G=$'\033[32m'; Y=$'\033[33m'; R=$'\033[31m'; B=$'\033[36m'; N=$'\033[0m'
else
  G=; Y=; R=; B=; N=
fi
info() { printf '%s[信息]%s %s\n' "$B" "$N" "$1"; }
ok()   { printf '%s[成功]%s %s\n' "$G" "$N" "$1"; }
warn() { printf '%s[注意]%s %s\n' "$Y" "$N" "$1"; }
err()  { printf '%s[错误]%s %s\n' "$R" "$N" "$1"; }

# ---------- 平台 ----------
# 本脚本同时支持 macOS 和 Linux：macOS 用 Docker Desktop（没有 sg/sudo/ss，也没有 WSL），
# Linux/WSL 保留原来的 docker 组回退逻辑。
IS_MACOS=0
[ "$(uname -s)" = "Darwin" ] && IS_MACOS=1

# ---------- docker 调用包装 ----------
DOCKER_MODE=""
_detect_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    err "没装 Docker。"
    if [ "$IS_MACOS" = "1" ]; then
      echo "  macOS：装 Docker Desktop   https://www.docker.com/products/docker-desktop/"
      echo "         或 brew install --cask docker"
    else
      echo "  Linux：sudo apt-get update && sudo apt-get install -y docker.io docker-compose-v2"
    fi
    exit 1
  fi
  if docker info >/dev/null 2>&1; then
    DOCKER_MODE="direct"
    return
  fi
  # macOS 上连不上基本只有一个原因：Docker Desktop 没启动。不需要 docker 组，也不用 sudo。
  if [ "$IS_MACOS" = "1" ]; then
    err "连不上 Docker，通常是 Docker Desktop 没启动。"
    echo "  启动：open -a Docker"
    echo "  等菜单栏鲸鱼图标变成运行中，再重跑本脚本。"
    exit 1
  fi
  # Linux：当前 shell 若未继承 docker 组（例如刚 usermod 过，或由脚本/IDE 拉起），自动退回 sg docker
  if command -v sg >/dev/null 2>&1 && sg docker -c "docker info" >/dev/null 2>&1; then
    DOCKER_MODE="sg"
  elif sudo -n docker info >/dev/null 2>&1; then
    DOCKER_MODE="sudo"
  else
    err "Docker 装好了但连不上/没权限。请确认后台服务在跑：sudo systemctl start docker"
    echo "  并把当前用户加入 docker 组（加完需重开终端）：sudo usermod -aG docker \$USER"
    exit 1
  fi
}

dq() { # 以合适权限执行 docker 命令
  case "$DOCKER_MODE" in
    direct) docker "$@" ;;
    sg)     sg docker -c "$(printf '%q ' docker "$@")" ;;
    sudo)   sudo -n docker "$@" ;;
  esac
}

# ---------- 准备配置 ----------
gen_env() {
  if [ -f .env ]; then
    info ".env 已存在，沿用其中的账号密码"
    return
  fi
  # 用 openssl 而不是 python3：macOS 12.3 起不再自带 python3，缺它时旧写法会静默
  # 生成空密码 + 空 JWT_SECRET 的 .env。
  if ! command -v openssl >/dev/null 2>&1; then
    err "没找到 openssl，无法生成 .env。"
    echo "  macOS：系统自带；异常时执行 xcode-select --install"
    echo "  Linux：sudo apt-get install -y openssl"
    exit 1
  fi
  warn "没找到 .env，正在从 .env.example 生成（随机密码）"
  local pwd_rand secret
  pwd_rand="$(openssl rand -hex 7)"      # 14 位
  secret="$(openssl rand -hex 32)"
  cat > .env <<EOF
ADMIN_USERNAME=admin
ADMIN_PASSWORD=$pwd_rand
JWT_SECRET=$secret
JWT_EXPIRE_HOURS=168
STORAGE_DRIVER=sqlite
DATABASE_DSN=/app/data/infinite-canvas.db
EOF
  ok "已生成 .env"
}

apply_patches() { # 幂等重放本地补丁（国内 GOPROXY + 服务端代拉模型列表）
  if [ ! -f patches/apply.py ]; then
    warn "patches/apply.py 不存在，跳过本地补丁"
    return 0
  fi
  # 补丁必须跑在宿主机上（改的是构建前的源码），所以宿主机得有 python3。
  if ! command -v python3 >/dev/null 2>&1; then
    err "没找到 python3，本地补丁跑不了（构建会缺 GOPROXY、服务端代拉模型列表等改动）。"
    echo "  macOS：xcode-select --install   或   brew install python3"
    echo "  Linux：sudo apt-get install -y python3"
    exit 1
  fi
  info "检查/重放本地补丁…"
  python3 patches/apply.py || { err "本地补丁应用失败"; exit 1; }
}

show_access() {
  local user pass
  user="$(grep -E '^ADMIN_USERNAME=' .env | cut -d= -f2- || echo admin)"
  pass="$(grep -E '^ADMIN_PASSWORD=' .env | cut -d= -f2- || echo '见 .env')"
  echo
  ok "无限画布已就绪"
  echo "  网页地址 : ${B}http://localhost:${HOST_PORT}${N}"
  # 备用地址只在 Linux/WSL 上有意义（Windows 侧 localhost 偶尔不通）。macOS 的
  # Docker Desktop 会把端口直接发布到 localhost，而且 BSD 的 hostname 没有 -I。
  if [ "$IS_MACOS" = "0" ]; then
    local host_ip
    host_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    if [ -n "$host_ip" ]; then
      echo "  备用地址 : ${B}http://${host_ip}:${HOST_PORT}${N}   (若 localhost 拒绝连接，用这个)"
    fi
  fi
  echo "  管理员账号: ${user}"
  echo "  管理员密码: ${pass}"
  echo "  数据目录 : ${APP_DIR}/data/infinite-canvas.db"
  echo "  查看日志 : ${B}./start.sh logs${N}"
}

wait_ready() { # 轮询等待后端就绪（/api/health 返回 ok），最多 180 秒
  local i body
  for i in $(seq 1 60); do
    body="$(curl -s -m 4 "http://127.0.0.1:${HOST_PORT}/api/health" 2>/dev/null || true)"
    if [ "$body" = "ok" ]; then
      ok "后端已就绪（第 ${i} 次探测，/api/health 返回 ok）"
      return 0
    fi
    sleep 3
  done
  warn "180 秒内后端仍未就绪，看日志排查：./start.sh logs"
  return 1
}

port_listening() { # 端口上是否有进程在监听（macOS 没有 ss，用 lsof；Linux 优先 ss）
  if command -v ss >/dev/null 2>&1 && ss -tln 2>/dev/null | grep -q ":${HOST_PORT} "; then
    return 0
  fi
  if command -v lsof >/dev/null 2>&1 && lsof -nP -iTCP:"${HOST_PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

port_busy_foreign() { # 端口被非本次容器占用则提示
  port_listening || return 1
  if dq ps --format '{{.Names}} {{.Ports}}' 2>/dev/null | grep -q "$CONTAINER"; then
    return 1
  fi
  return 0
}

# ---------- 各子命令 ----------
do_start() {
  _detect_docker
  gen_env
  mkdir -p data
  printf '%s\n' "========================================================" \
                "  无限画布 infinite-canvas 一键启动" \
                "  用法: ./start.sh [start|stop|restart|rebuild|logs|status|update]" \
                "========================================================"

  if port_busy_foreign; then
    err "宿主机端口 ${HOST_PORT} 已被别的程序占用（不是本容器）。"
    if [ "$IS_MACOS" = "1" ]; then
      echo "  查看占用：lsof -nP -iTCP:${HOST_PORT} -sTCP:LISTEN"
    else
      echo "  查看占用：ss -tlnp | grep :${HOST_PORT}"
    fi
    echo "  改端口  ：编辑 start.sh 顶部的 HOST_PORT 与 ${COMPOSE_FILE} 的端口映射"
    exit 1
  fi

  if ! dq image inspect "$IMAGE" >/dev/null 2>&1; then
    apply_patches
    warn "本地还没有镜像 $IMAGE，开始构建（首次约 3~10 分钟，含本机缓存）"
    if ! dq compose -f "$COMPOSE_FILE" build; then
      err "构建失败。国内网络常见原因：容器内 proxy.golang.org 不可达，"
      echo "  Dockerfile 需保持 ENV GOPROXY=https://goproxy.cn,direct"
      exit 1
    fi
    ok "镜像构建完成"
  else
    info "镜像已存在，跳过构建（要重新构建用 ./start.sh rebuild）"
  fi

  info "启动容器…"
  dq compose -f "$COMPOSE_FILE" up -d || { err "启动失败"; exit 1; }
  wait_ready || true
  show_access
}

do_stop() {
  _detect_docker
  info "停止容器…"
  dq compose -f "$COMPOSE_FILE" down
  ok "已停止（数据保留在 ${APP_DIR}/data）"
}

do_restart() {
  do_stop >/dev/null 2>&1 || true
  do_start
}

do_rebuild() {
  _detect_docker
  gen_env
  apply_patches
  info "重新构建镜像（走 Docker 层缓存，改代码后通常 1~3 分钟）…"
  dq compose -f "$COMPOSE_FILE" build || { err "构建失败"; exit 1; }
  dq compose -f "$COMPOSE_FILE" up -d
  wait_ready || true
  show_access
}

do_logs() {
  _detect_docker
  dq logs -f --tail 100 "$CONTAINER"
}

do_status() {
  _detect_docker
  echo "--- 容器 ---"
  dq compose -f "$COMPOSE_FILE" ps
  echo "--- 端口 ---"
  if port_listening; then
    echo "  端口 ${HOST_PORT} 已监听"
  else
    echo "  端口 ${HOST_PORT} 未监听"
  fi
  echo "--- 网页 ---"
  printf '  http://localhost:%s/  ->  HTTP %s\n' "$HOST_PORT" \
    "$(curl -s -o /dev/null -m 5 -w '%{http_code}' "http://127.0.0.1:${HOST_PORT}/" 2>/dev/null || echo '无响应')"
  echo "--- 账号 ---"
  echo "  $(grep -E '^ADMIN_USERNAME=' .env 2>/dev/null || echo 'ADMIN_USERNAME=admin')  /  $(grep -E '^ADMIN_PASSWORD=' .env 2>/dev/null | cut -d= -f2- || echo '?')"
  echo "--- 镜像 ---"
  dq images "$IMAGE" --format '  {{.Repository}}:{{.Tag}}  {{.Size}}  构建于 {{.CreatedSince}}'
}

do_update() {
  _detect_docker
  warn "本目录是 ghfast.top 镜像下载的源码包（无 git 历史），更新 = 重新下载覆盖。"
  read -r -p "会保留 .env 与 data/，继续？[y/N] " ans
  [ "$ans" = "y" ] || [ "$ans" = "Y" ] || { info "已取消"; exit 0; }
  local tmp="/tmp/ic-update-$$.tar.gz"
  info "下载最新源码…"
  curl -fsSL --max-time 300 -o "$tmp" \
    "https://ghfast.top/https://github.com/tigerowo/infinite-canvas/archive/refs/heads/main.tar.gz" \
    || { err "下载失败"; exit 1; }
  local bak="${APP_DIR}/../infinite-canvas.bak-$(date +%Y%m%d-%H%M%S)"
  cp -r "$APP_DIR" "$bak" && info "旧版本已备份到 $bak"
  tar xzf "$tmp" -C "$APP_DIR" --strip-components=1
  rm -f "$tmp"
  # 更新后本地补丁要重新打（Dockerfile GOPROXY + 服务端代拉模型列表）
  apply_patches
  do_rebuild
}

case "${1:-start}" in
  start)   do_start ;;
  stop)    do_stop ;;
  restart) do_restart ;;
  rebuild) do_rebuild ;;
  logs)    do_logs ;;
  status)  do_status ;;
  update)  do_update ;;
  help|-h|--help)
    sed -n '2,4p' "$0" | sed 's/^# \{0,1\}//'
    ;;
  *)
    err "未知参数：$1"; sed -n '2,4p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
esac
