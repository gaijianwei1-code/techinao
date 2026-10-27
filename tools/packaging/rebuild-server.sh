#!/usr/bin/env bash
set -euo pipefail

SRC=/home/gai/ic-build/src
PROJECT=/mnt/d/.Hermes/infinite-canvas

echo "== resync source (drops data/.env/node_modules/.next) =="
cd "$PROJECT"
tar --exclude=./data --exclude=./.env --exclude=./web/node_modules --exclude=./web/.next -cf - . | tar xf - -C "$SRC"
ls -l "$SRC/service/prompt_cache.go" "$SRC/main.go" "$SRC/repository/prompt.go"

echo "== rebuild server.exe (windows/amd64) =="
docker run --rm \
  -v "$SRC":/src \
  -v /home/gai/ic-build/out:/out \
  -w /src \
  -e GOPROXY=https://goproxy.cn,direct \
  -e GOSUMDB=sum.golang.google.cn \
  -e CGO_ENABLED=0 -e GOOS=windows -e GOARCH=amd64 \
  golang:1.25-alpine sh -c "go vet ./service/... ./repository/... . ; go build -trimpath -ldflags '-s -w' -o /out/server.exe ."
ls -lh /home/gai/ic-build/out/server.exe
echo "== done =="
