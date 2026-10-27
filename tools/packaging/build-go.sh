#!/usr/bin/env bash
set -euo pipefail

echo "== go cross build (windows/amd64) =="
mkdir -p /home/gai/ic-build/out
docker run --rm \
  -v /home/gai/ic-build/src:/src \
  -v /home/gai/ic-build/out:/out \
  -w /src \
  -e GOPROXY=https://goproxy.cn,direct \
  -e GOSUMDB=sum.golang.google.cn \
  -e CGO_ENABLED=0 -e GOOS=windows -e GOARCH=amd64 \
  golang:1.25-alpine sh -c "go build -trimpath -ldflags '-s -w' -o /out/server.exe ."
ls -lh /home/gai/ic-build/out
echo "== go build done =="
