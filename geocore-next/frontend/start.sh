#!/bin/bash
set -e

export PORT=5000
export BASE_PATH="/"
export BACKEND_PORT="${BACKEND_PORT:-9000}"

cd /home/runner/workspace/geocore-next/frontend

echo "[geocore-web] Installing dependencies if needed..."
pnpm install --frozen-lockfile 2>/dev/null || pnpm install

echo "[geocore-web] Starting Vite dev server on port $PORT..."
exec pnpm --filter @workspace/web run dev
