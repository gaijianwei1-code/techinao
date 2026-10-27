#!/usr/bin/env bash
set -euo pipefail

echo "== build launcher.exe =="
docker run --rm \
  -v /mnt/c/Users/11/.trae-cn/work/6aaa7312573ee181ecf93750/ic-build/launcher:/src \
  -v /home/gai/ic-build/out:/out \
  -w /src \
  -e GOPROXY=https://goproxy.cn,direct \
  -e GOSUMDB=sum.golang.google.cn -e CGO_ENABLED=0 -e GOOS=windows -e GOARCH=amd64 \
  golang:1.25-alpine sh -c "go mod init iclauncher >/dev/null 2>&1 || true; go vet ./... ; go build -trimpath -ldflags '-H=windowsgui -s -w' -o /out/launcher.exe ."
ls -lh /home/gai/ic-build/out
echo "== launcher done =="
