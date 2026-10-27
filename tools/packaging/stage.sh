#!/usr/bin/env bash
set -euo pipefail

SRC=/home/gai/ic-build/src
STAGE=/home/gai/ic-build/stage
OUT=/home/gai/ic-build/out

rm -rf "$STAGE"
mkdir -p "$STAGE/web/.next/static"

# Next.js standalone 运行时
cp -r "$SRC/web/.next/standalone/." "$STAGE/web/"
cp -r "$SRC/web/.next/static/." "$STAGE/web/.next/static/"
cp -r "$SRC/web/public" "$STAGE/web/public"
cp "$OUT/server.exe" "$STAGE/server.exe"

echo "--- symlink check ---"
find "$STAGE" -type l | wc -l

echo "--- stage size ---"
du -sh "$STAGE" "$STAGE/web" "$STAGE/web/public" "$STAGE/web/.next"

echo "--- pack ---"
mkdir -p /mnt/c/Users/11/.trae-cn/work/6aaa7312573ee181ecf93750/ic-build/downloads
tar czf /mnt/c/Users/11/.trae-cn/work/6aaa7312573ee181ecf93750/ic-build/downloads/stage.tar.gz -C "$STAGE" .
ls -lh /mnt/c/Users/11/.trae-cn/work/6aaa7312573ee181ecf93750/ic-build/downloads/stage.tar.gz
echo "== stage done =="
