#!/bin/bash
  # ─────────────────────────────────────────────────────────────────────────────
  # GeoCore database backup — runs daily via Kubernetes CronJob
  # Backs up to Cloudflare R2 (S3-compatible)
  # ─────────────────────────────────────────────────────────────────────────────
  set -euo pipefail

  TIMESTAMP=$(date +%Y%m%d_%H%M%S)
  BACKUP_FILE="geocore_backup_${TIMESTAMP}.sql"
  S3_BUCKET="s3://geocore-backups/db"

  echo "[backup] Starting backup at ${TIMESTAMP}"

  pg_dump \
    -h "${DB_HOST}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    --no-password \
    --format=custom \
    --compress=9 \
    -f "/tmp/${BACKUP_FILE}"

  echo "[backup] Uploading to ${S3_BUCKET}..."
  aws s3 cp "/tmp/${BACKUP_FILE}" "${S3_BUCKET}/${BACKUP_FILE}" \
    --endpoint-url "https://${CLOUDFLARE_ACCOUNT_ID}.r2.cloudflarestorage.com"

  # Delete local temp file
  rm -f "/tmp/${BACKUP_FILE}"

  # Prune backups older than 30 days
  aws s3 ls "${S3_BUCKET}/" \
    --endpoint-url "https://${CLOUDFLARE_ACCOUNT_ID}.r2.cloudflarestorage.com" | \
    awk '{print $4}' | while read -r key; do
      FILEDATE=$(echo "${key}" | grep -oP '[0-9]{8}')
      if [[ -n "${FILEDATE}" && "${FILEDATE}" < "$(date -d '30 days ago' +%Y%m%d)" ]]; then
        aws s3 rm "${S3_BUCKET}/${key}" \
          --endpoint-url "https://${CLOUDFLARE_ACCOUNT_ID}.r2.cloudflarestorage.com"
        echo "[backup] Pruned: ${key}"
      fi
    done

  echo "[backup] Done."
  