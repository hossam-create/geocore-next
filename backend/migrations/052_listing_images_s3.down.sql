-- 052 down: Remove S3 image columns from listing_images

-- Remove new columns
ALTER TABLE listing_images
    DROP COLUMN IF EXISTS group_id,
    DROP COLUMN IF EXISTS image_id,
    DROP COLUMN IF EXISTS width,
    DROP COLUMN IF EXISTS height,
    DROP COLUMN IF EXISTS bytes,
    DROP COLUMN IF EXISTS mime_type,
    DROP COLUMN IF EXISTS variant,
    DROP COLUMN IF EXISTS created_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_listing_images_group_id;
DROP INDEX IF EXISTS idx_listing_images_image_id;
DROP INDEX IF EXISTS idx_listing_images_variant;
