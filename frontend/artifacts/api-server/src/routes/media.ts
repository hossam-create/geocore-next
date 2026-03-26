import { Router, Request, Response } from "express";
import { S3Client, PutObjectCommand, DeleteObjectCommand } from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
import crypto from "crypto";

const router = Router();

// ── R2 / S3 Client ──────────────────────────────────────────────────────────

const R2_ENDPOINT = process.env.CLOUDFLARE_R2_ENDPOINT;
const R2_ACCESS_KEY = process.env.CLOUDFLARE_R2_ACCESS_KEY;
const R2_SECRET_KEY = process.env.CLOUDFLARE_R2_SECRET_KEY;
const R2_BUCKET = process.env.CLOUDFLARE_R2_BUCKET || "geocore-media";
const R2_PUBLIC_URL = process.env.CLOUDFLARE_R2_PUBLIC_URL || `https://media.geocore.app`;

const R2_CONFIGURED = !!(R2_ENDPOINT && R2_ACCESS_KEY && R2_SECRET_KEY);

let s3Client: S3Client | null = null;
if (R2_CONFIGURED) {
  s3Client = new S3Client({
    region: "auto",
    endpoint: R2_ENDPOINT,
    credentials: {
      accessKeyId: R2_ACCESS_KEY!,
      secretAccessKey: R2_SECRET_KEY!,
    },
  });
}

// ── Allowed types ────────────────────────────────────────────────────────────

const ALLOWED_TYPES: Record<string, string> = {
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/webp": "webp",
  "image/gif": "gif",
  "image/avif": "avif",
};

const MAX_SIZE_BYTES = 10 * 1024 * 1024; // 10MB

// ── POST /media/upload-url — get presigned upload URL ──────────────────────

router.post("/upload-url", async (req: Request, res: Response) => {
  const { filename, content_type, folder = "listings", size } = req.body;

  if (!filename || !content_type) {
    return res.status(400).json({ success: false, message: "filename and content_type are required" });
  }

  if (!ALLOWED_TYPES[content_type]) {
    return res.status(400).json({
      success: false,
      message: `Invalid file type. Allowed: ${Object.keys(ALLOWED_TYPES).join(", ")}`,
    });
  }

  if (size && size > MAX_SIZE_BYTES) {
    return res.status(400).json({ success: false, message: "File too large. Maximum size is 10MB" });
  }

  const ext = ALLOWED_TYPES[content_type];
  const uniqueId = crypto.randomBytes(12).toString("hex");
  const safeFolder = folder.replace(/[^a-z0-9_-]/gi, "");
  const key = `${safeFolder}/${uniqueId}.${ext}`;
  const publicUrl = `${R2_PUBLIC_URL}/${key}`;

  // ── Real R2 presigned URL ──────────────────────────────────────────────────
  if (R2_CONFIGURED && s3Client) {
    try {
      const command = new PutObjectCommand({
        Bucket: R2_BUCKET,
        Key: key,
        ContentType: content_type,
        CacheControl: "public, max-age=31536000, immutable",
        Metadata: {
          "original-filename": filename.substring(0, 100),
          "uploaded-by": "geocore-api",
        },
      });

      const uploadUrl = await getSignedUrl(s3Client, command, { expiresIn: 300 }); // 5 min

      return res.json({
        success: true,
        data: {
          upload_url: uploadUrl,
          public_url: publicUrl,
          key,
          expires_in: 300,
          max_size_bytes: MAX_SIZE_BYTES,
        },
      });
    } catch (err) {
      return res.status(500).json({ success: false, message: "Failed to generate upload URL" });
    }
  }

  // ── Mock response (R2 not configured) ─────────────────────────────────────
  return res.json({
    success: true,
    data: {
      upload_url: `https://mock-r2.dev/upload/${key}?token=mock`,
      public_url: `https://picsum.photos/seed/${uniqueId}/800/600`,
      key,
      expires_in: 300,
      max_size_bytes: MAX_SIZE_BYTES,
      _mock: true,
    },
  });
});

// ── DELETE /media/delete — remove file from R2 ────────────────────────────

router.delete("/delete", async (req: Request, res: Response) => {
  const { key } = req.body;

  if (!key || typeof key !== "string" || key.includes("..")) {
    return res.status(400).json({ success: false, message: "Invalid key" });
  }

  if (R2_CONFIGURED && s3Client) {
    try {
      await s3Client.send(new DeleteObjectCommand({ Bucket: R2_BUCKET, Key: key }));
      return res.json({ success: true, message: "File deleted" });
    } catch {
      return res.status(500).json({ success: false, message: "Failed to delete file" });
    }
  }

  return res.json({ success: true, message: "File deleted (mock)" });
});

// ── GET /media/config — expose upload config to frontend ──────────────────

router.get("/config", (_req: Request, res: Response) => {
  res.json({
    success: true,
    data: {
      max_size_bytes: MAX_SIZE_BYTES,
      max_size_mb: MAX_SIZE_BYTES / (1024 * 1024),
      allowed_types: Object.keys(ALLOWED_TYPES),
      r2_configured: R2_CONFIGURED,
      public_url_base: R2_PUBLIC_URL,
    },
  });
});

export default router;
