#!/bin/bash
set -e

BACKEND_PORT="${BACKEND_PORT:-9000}"

echo "[geocore-backend] Starting Redis..."
redis-server --daemonize yes --logfile /tmp/redis.log 2>/dev/null || echo "[geocore-backend] Redis already running or failed (ok)"

echo "[geocore-backend] Killing any process on port $BACKEND_PORT..."
fuser -k "${BACKEND_PORT}/tcp" 2>/dev/null || true
sleep 1

cd /home/runner/workspace/geocore-next/backend

echo "[geocore-backend] Compiling and starting Go API on port $BACKEND_PORT..."
exec go run ./cmd/api/
