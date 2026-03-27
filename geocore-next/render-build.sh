#!/bin/bash
set -euo pipefail

echo "==> Installing pnpm..."
npm install -g pnpm@9 --silent

echo "==> Building Web Frontend..."
cd frontend/artifacts/web
pnpm install --frozen-lockfile
PORT=3000 BASE_PATH=/ BACKEND_PORT=10000 NODE_ENV=production pnpm run build
cd ../../..

echo "==> Building Admin Frontend..."
cd frontend/artifacts/admin
pnpm install --frozen-lockfile
PORT=3001 BASE_PATH=/ BACKEND_PORT=10000 NODE_ENV=production pnpm run build 2>/dev/null || echo "[warn] admin build skipped or failed — continuing"
cd ../../..

echo "==> Copying frontend dist to backend/web..."
rm -rf backend/web backend/admin
mkdir -p backend/web backend/admin
# web output is dist/public
if [ -d frontend/artifacts/web/dist/public ]; then
  cp -r frontend/artifacts/web/dist/public/. backend/web/
else
  cp -r frontend/artifacts/web/dist/. backend/web/
fi
# admin output
if [ -d frontend/artifacts/admin/dist/public ]; then
  cp -r frontend/artifacts/admin/dist/public/. backend/admin/
elif [ -d frontend/artifacts/admin/dist ]; then
  cp -r frontend/artifacts/admin/dist/. backend/admin/
fi

echo "==> Compiling Go binary..."
cd backend
go build -ldflags="-s -w" -o ../server ./cmd/api/
cd ..

echo "==> Build complete!"
