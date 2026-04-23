-- ════════════════════════════════════════════════════════════════════════════
-- 052: Upgrade listing_images for S3-based image system
-- Adds: width, height, bytes, mime_type, variant, group_id, image_id, created_at
-- Migrates legacy TEXT[] images from listings.images column
-- ════════════════════════════════════════════════════════════════════════════

-- 1. Add new columns to listing_images (backward compatible — all have defaults)
ALTER TABLE listing_images
    ADD COLUMN IF NOT EXISTS group_id   UUID,
    ADD COLUMN IF NOT EXISTS image_id   UUID,
    ADD COLUMN IF NOT EXISTS width      INTEGER      DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height     INTEGER      DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bytes      BIGINT       DEFAULT 0,
    ADD COLUMN IF NOT EXISTS mime_type  VARCHAR(50)  DEFAULT 'image/jpeg',
    ADD COLUMN IF NOT EXISTS variant    VARCHAR(20)  DEFAULT 'large',
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ  DEFAULT NOW();

-- 2. Add indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_listing_images_group_id   ON listing_images (group_id);
CREATE INDEX IF NOT EXISTS idx_listing_images_image_id   ON listing_images (image_id);
CREATE INDEX IF NOT EXISTS idx_listing_images_variant    ON listing_images (variant);

-- 3. Migrate legacy TEXT[] images from listings.images → listing_images
-- Only migrates listings that have images in the TEXT[] column but no listing_images rows yet.
-- This is idempotent — safe to run multiple times.
INSERT INTO listing_images (id, listing_id, url, variant, sort_order, is_cover, mime_type, created_at)
SELECT
    gen_random_uuid(),
    l.id,
    UNNEST(l.images),
    'original',
    ordinality - 1,
    (ordinality = 1),
    'image/jpeg',
    NOW()
FROM listings l
CROSS JOIN LATERAL UNNEST(l.images) WITH ORDINALITY AS u(url, ordinality)
WHERE l.images IS NOT NULL
  AND array_length(l.images, 1) > 0
  AND NOT EXISTS (
      SELECT 1 FROM listing_images li WHERE li.listing_id = l.id
  );

-- 4. Update variant for existing listing_images that were created from the Upload endpoint
-- (they already have URLs pointing to R2 — mark as 'large' variant)
UPDATE listing_images
SET variant  = 'large',
    mime_type = CASE
        WHEN url LIKE '%.webp' THEN 'image/webp'
        WHEN url LIKE '%.png'  THEN 'image/png'
        ELSE 'image/jpeg'
    END
WHERE variant = 'large'
  AND width = 0
  AND url LIKE 'http%';

-- 5. Add comment for documentation
COMMENT ON COLUMN listing_images.variant IS 'Image size variant: thumbnail (200px), medium (600px), large (1200px), original (4096px)';
COMMENT ON COLUMN listing_images.group_id IS 'Groups size variants from a single upload (references images.group_id)';
COMMENT ON COLUMN listing_images.image_id IS 'References the images table for full variant lookup';
