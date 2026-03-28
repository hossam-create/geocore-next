#!/bin/bash
set -euo pipefail

# ── Node / pnpm setup ─────────────────────────────────────────────────────────
if ! command -v pnpm &>/dev/null; then
  echo "==> Installing pnpm..."
  npm install -g pnpm@9 --silent
fi
echo "pnpm $(pnpm --version)"

# ── Web Frontend ──────────────────────────────────────────────────────────────
echo "==> Building Web Frontend..."
cd frontend/artifacts/web
pnpm install --no-frozen-lockfile
PORT=3000 BASE_PATH=/ BACKEND_PORT=10000 NODE_ENV=production pnpm run build
cd ../../..

# ── Admin Frontend (optional — skip on failure) ───────────────────────────────
echo "==> Building Admin Frontend..."
if [ -f frontend/artifacts/admin/package.json ]; then
  (
    cd frontend/artifacts/admin
    pnpm install --no-frozen-lockfile
    PORT=3001 BASE_PATH=/ BACKEND_PORT=10000 NODE_ENV=production pnpm run build
  ) && echo "Admin build OK" || echo "[warn] Admin build failed — skipping"
fi

# ── Copy dist → backend/web ───────────────────────────────────────────────────
echo "==> Copying frontend dist..."
rm -rf backend/web backend/admin
mkdir -p backend/web backend/admin

# web output: dist/public
WEB_DIST="frontend/artifacts/web/dist/public"
[ -d "$WEB_DIST" ] || WEB_DIST="frontend/artifacts/web/dist"
cp -r "$WEB_DIST"/. backend/web/

# admin output (best-effort)
ADMIN_DIST="frontend/artifacts/admin/dist/public"
[ -d "$ADMIN_DIST" ] || ADMIN_DIST="frontend/artifacts/admin/dist"
[ -d "$ADMIN_DIST" ] && cp -r "$ADMIN_DIST"/. backend/admin/ || true

# ── Go binary ─────────────────────────────────────────────────────────────────
echo "==> Compiling Go binary..."
cd backend
go build -ldflags="-s -w" -o ../server ./cmd/api/
cd ..

echo "==> Build complete!"
