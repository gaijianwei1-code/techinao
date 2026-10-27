---
title: 迁移到 macOS
description: 把整个项目（含数据和配置）从 Windows/WSL 搬到 Mac
---

# 迁移到 macOS

项目本体（Dockerfile、compose、Go 后端、Next.js 前端、`patches/`）本来就是跨平台的，Mac 上直接能用。需要搬的只有**代码目录 + 数据 + `.env`**，需要改的只有几个原先绑死 Linux 的外围脚本——那些已经在 `start.sh` / `scripts/tunnel/tunnel.sh` 里按 `uname -s` 做了分支。

## 一、Mac 上先装两样东西

1. **Docker Desktop**（或 OrbStack / Colima，任一提供 `docker` 命令的运行时）
   ```bash
   brew install --cask docker     # 或去官网下载安装
   open -a Docker                 # 启动，等菜单栏鲸鱼图标变成运行中
   ```
   Dockerfile 用的三个基础镜像（`oven/bun`、`golang`、`node`）都有 arm64，Apple Silicon 原生跑，不需要 Rosetta。

2. **python3**（构建时 `patches/apply.py` 要跑在宿主机上，改的是构建前的源码）
   ```bash
   xcode-select --install         # 装命令行工具即带 python3
   # 或 brew install python3
   ```

## 二、要拷什么

**项目不是 git 仓库，只能整目录拷。** 但有几样不用带：

| 路径 | 大小 | 拷不拷 |
| --- | --- | --- |
| `.env` | 几百字节 | **必须拷**。里面有 `ADMIN_PASSWORD` 和 `JWT_SECRET` |
| `data/infinite-canvas.db` | 551 MB | **必须拷**。全部用户、素材、生成记录、渠道配置都在这 |
| `data/prompts/` | 58 MB | **必须拷**。提示词库与封面缓存 |
| `data/logs/` | 3.6 MB | 建议拷，是 AI 调用日志 |
| `dist/` | 54 MB | **不用拷**。Windows 安装包（`.exe`），Mac 上没用 |
| `logs/`、`.run/` | 很小 | 不用拷。运行期日志和隧道 pid/地址 |
| `web/.next`、`web/node_modules` | — | 不用拷。构建产物，Mac 上会重新生成 |

> `.env` 里的 `JWT_SECRET` **不要重新生成**，否则所有已登录用户（包括你）都会被登出。`gen_env` 只在 `.env` 不存在时才生成新的，拷过去就会沿用。

## 三、步骤

```bash
# 1. 拷到 Mac（放在 ~ 下，见下面「必须注意」）
#    用移动硬盘 / 局域网 scp / rsync 都行，注意把 data/ 和 .env 一起带上
rsync -av --exclude 'dist' --exclude 'logs' --exclude '.run' \
      --exclude 'web/.next' --exclude 'web/node_modules' \
      /path/to/infinite-canvas/ mac:~/infinite-canvas/

# 或者直接拷一个 tar.gz 包，解压即可（推荐：tar 会保留可执行位）
tar -xzf infinite-canvas-mac-*.tar.gz -C ~/

# 2. 补回可执行位
#    用 tar.gz 解压的话这步可以跳过（权限已经带上了）；用网盘/zip/资源管理器拷贝的必须做，
#    因为项目不是 git 仓库，可执行位不会跟着走。
cd ~/infinite-canvas
ls -l start.sh             # 应显示 -rwxr-xr-x，不是的话：
chmod +x start.sh scripts/tunnel/tunnel.sh "一键启动全部服务.command"

# 3. 启动
./start.sh start          # 或双击「一键启动全部服务.command」
```

> 打包时用 `tar` 而不是 `zip`：tar 会保留可执行位，而且文件名按原始字节存——Windows 的 zip 工具常把中文名写成 GBK，macOS 解出来会变乱码（`一键启动全部服务.command` 这种文件名首当其冲）。

首次会构建镜像（3~10 分钟）。之后：

```bash
./start.sh rebuild        # 改了代码后重建
./start.sh logs           # 看日志
./start.sh status         # 看状态
./start.sh stop           # 停掉
```

**想继续给外地朋友用**，装好 cloudflared 再起隧道：

```bash
brew install cloudflared
./scripts/tunnel/tunnel.sh cloudflared     # 或直接双击「一键启动全部服务.command」
```

## 四、迁移后自检

- [ ] `curl http://127.0.0.1:3030/api/health` 返回 `ok`
- [ ] 浏览器打开 `http://localhost:3030` 能看到页面
- [ ] 用原来的账号密码能登录（登录不了多半是 `.env` 里的密码和实际不一致，见下面「已知差异」）
- [ ] 「我的素材」和生成历史还在
- [ ] 后台「系统设置 - 私有配置 - 渠道」里渠道还在、能拉到模型
- [ ] 跑一次生图，能出图

## 五、必须注意

**项目要放在 `~/` 下。** Docker Desktop 默认只把 `/Users`、`/Volumes`、`/tmp`、`/private` 共享给容器。放外置硬盘、iCloud Drive、网络盘会报 `Mounts denied: ... is not shared`。放 `~/infinite-canvas` 最省事。

**端口是 3030**，和 Windows 上一样。macOS 的 AirPlay 接收器占的是 5000/7000，不冲突。

## 六、已知差异

**Windows 专用文件在 Mac 上是死的**，不会被触发，留着不影响：6 个 `.bat`、`keepalive.ps1`、`tools/packaging/`、`tools/launcher/`、`tools/envcheck/`、`dist/*.exe`。

**不需要 keepalive。** Windows 上要 `keepalive.ps1` 是因为 WSL2 空闲会把整个虚拟机关掉，连带容器一起停。macOS 上 Docker Desktop 是常驻的，没这个问题。

**统一用 `docker-compose.local.yml`。** 另一个 `docker-compose.yml` 是给 ghcr 镜像部署用的，端口写的是 3000、容器名也固定成 `infinite-canvas`，和 `start.sh` 假设的 3030 / `infinite-canvas-app-1` 对不上。`start.sh` 用的就是 `.local.yml`，别手动 `docker compose up` 另一个。

**`.env` 里的管理员密码可能已经过期。** 如果 `.env` 里的 `ADMIN_PASSWORD` 和你在界面上实际改过的密码不一致，`start.sh status` 会显示错的密码，`.env` 也不会影响登录（登录校验的是数据库里的哈希）。要不要同步看你，不影响运行。

## 七、排错

| 现象 | 处理 |
| --- | --- |
| `Mounts denied` | 项目没放在 Docker Desktop 的共享目录里，挪到 `~/` 下 |
| `连不上 Docker，通常是 Docker Desktop 没启动` | `open -a Docker`，等菜单栏图标变运行中 |
| `没找到 python3，本地补丁跑不了` | `xcode-select --install` |
| `database is locked` | macOS 上 SQLite 跑在绑定挂载上偶发；先 `./start.sh restart`，反复出现就改用 OrbStack，或把 `docker-compose.local.yml` 的 `./data` 换成具名 volume（记得先把数据迁进去） |
| 构建卡在拉依赖 | Dockerfile 里已把 Go/npm/Debian 源都换成国内镜像；换到境外网络时这些镜像反而慢，可按需改回官方源 |
| 隧道 502 | 先确认 `curl http://127.0.0.1:3030/api/health` 是 `ok`；隧道脚本已改用 `127.0.0.1`（macOS 的 localhost 常解析到 `::1`，Docker Desktop 的 IPv6 转发不稳） |
| 双击 `.command` 没反应 | 多半是换行符变成了 CRLF（shebang 会带上 `\r`），用 `file 一键启动全部服务.command` 确认，`sed -i '' 's/\r$//'` 修回 LF |

## 八、可选：拷贝前给数据库瘦身

数据库 551 MB 里约 382 MB 是 base64 图片（生成日志 247 MB + 任务表 135 MB）。先按最新代码重建、打开一次生图工作台触发日志瘦身，再回收空间，拷贝量能小一大截：

```bash
./start.sh rebuild
# 浏览器打开一次生图工作台，等历史记录加载完（会把日志里的 base64 挪进本地存储并推回服务端）
docker run --rm -v "$PWD/data:/data" alpine sh -c "apk add --no-cache sqlite && sqlite3 /data/infinite-canvas.db 'VACUUM;'"
```

不做也不影响使用——SQLite 不会自动回收已删除内容占的空间，所以文件大小不会自己变小，但功能没问题。
