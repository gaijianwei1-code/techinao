#!/usr/bin/env bash
set -euo pipefail

# 编译环境检测工具（tools/envcheck）。
#
# 为什么钉住 Go 1.20：Go 1.21 起最低要求 Windows 10，
# 而 1.20 是最后一个支持 Windows 7 / 8 / Server 2008R2 / 2012 的版本，
# 朋友的老机器要靠它才能跑起来给出结论。
#
# 基础镜像用本地已有的 golang:1.25-alpine（Docker Hub 拉取太慢），
# 在容器里另外下载 Go 1.20 工具链并用绝对路径调用，
# 避免镜像自带的 go1.25 抢先命中 PATH，导致产物在 Win7 上闪退。
#
# 用法（在 WSL 里执行）：bash tools/packaging/build-envcheck.sh
PROJECT=/mnt/d/.Hermes/infinite-canvas
OUT=/home/gai/ic-build/out
BASE_IMAGE=golang:1.25-alpine
GO120=/opt/go120/bin/go
GO120_VERSION=1.20.14
GO120_MIRROR=https://mirrors.aliyun.com/golang
XTEXT=v0.14.0   # 与 Go 1.20 兼容的 x/text 版本

mkdir -p "$OUT"

docker run --rm \
  -v "$PROJECT/tools/envcheck":/src:ro \
  -v "$OUT":/out \
  -w /work \
  -e GOPROXY=https://goproxy.cn,direct -e GOSUMDB=sum.golang.google.cn \
  -e CGO_ENABLED=0 -e GOOS=windows -e GOARCH=amd64 -e GOFLAGS=-mod=mod \
  "$BASE_IMAGE" sh -c "
    set -e
    apk add --no-cache binutils curl tar >/dev/null

    echo '== 准备 Go $GO120_VERSION 工具链 =='
    curl -fsSL $GO120_MIRROR/go$GO120_VERSION.linux-amd64.tar.gz -o /tmp/go120.tar.gz
    mkdir -p /opt/go120 && tar -C /opt/go120 --strip-components=1 -xzf /tmp/go120.tar.gz
    export GOROOT=/opt/go120
    $GO120 version

    # 在容器内的工作副本里编译，不写回项目目录
    cp -r /src /work
    $GO120 get golang.org/x/text@$XTEXT
    $GO120 mod tidy
    $GO120 vet ./...
    $GO120 build -trimpath -ldflags '-s -w' -o /out/ic-env-check.exe .
    GOARCH=386 $GO120 build -trimpath -ldflags '-s -w' -o /out/ic-env-check-x86.exe .

    echo '== 产物编译版本（必须是 go1.20，否则 Win7 上会闪退）=='
    $GO120 version /out/ic-env-check.exe
    $GO120 version /out/ic-env-check-x86.exe
    for f in /out/ic-env-check.exe /out/ic-env-check-x86.exe; do
      $GO120 version \$f | grep -q 'go1.20' || { echo \"编译版本不对：\$f\"; exit 1; }
    done

    echo '== 导入的 DLL（只应有 kernel32.dll）=='
    objdump -p /out/ic-env-check.exe | grep -i 'DLL Name' | sort -u
    objdump -p /out/ic-env-check-x86.exe | grep -i 'DLL Name' | sort -u
  "

ls -lh "$OUT"/ic-env-check*.exe
echo "== envcheck build done =="
