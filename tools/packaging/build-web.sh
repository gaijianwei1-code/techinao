#!/usr/bin/env bash
set -euo pipefail

echo "== web build start =="
docker run --rm \
  -v /home/gai/ic-build/src:/app \
  -w /app/web \
  -e BUN_CONFIG_REGISTRY=https://registry.npmmirror.com \
  -e NEXT_TELEMETRY_DISABLED=1 \
  oven/bun:1.3.14 sh -c 'bun install --frozen-lockfile && bun run build'
echo "== web build done =="
