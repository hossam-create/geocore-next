-- 054 down: Remove search quality improvements

-- Drop functions
DROP FUNCTION IF EXISTS expand_query_with_synonyms(TEXT, TEXT);
DROP FUNCTION IF EXISTS update_popular_query(TEXT, TEXT, INT, TEXT);

-- Drop columns from search_queries
ALTER TABLE search_queries 
    DROP COLUMN IF EXISTS filters_json,
    DROP COLUMN IF EXISTS position_in_results,
    DROP COLUMN IF EXISTS clicked_listing_id,
    DROP COLUMN IF EXISTS zero_result,
    DROP COLUMN IF EXISTS query_hash;

-- Drop columns from listings
ALTER TABLE listings 
    DROP COLUMN IF EXISTS last_viewed_at,
    DROP COLUMN IF EXISTS last_searched_at,
    DROP COLUMN IF EXISTS search_impression_count,
    DROP COLUMN IF EXISTS search_click_count,
    DROP COLUMN IF EXISTS view_count;

-- Drop listing analytics tables
DROP TABLE IF EXISTS listing_clicks;
DROP TABLE IF EXISTS listing_views;

-- Drop indexes
DROP INDEX IF EXISTS idx_listings_last_viewed;
DROP INDEX IF EXISTS idx_listings_last_searched;
DROP INDEX IF EXISTS idx_listings_search_click_count;
DROP INDEX IF EXISTS idx_listings_view_count;
DROP INDEX IF EXISTS idx_search_queries_clicked;
DROP INDEX IF EXISTS idx_search_queries_zero_result;
DROP INDEX IF EXISTS idx_search_queries_query_hash;
DROP INDEX IF EXISTS idx_search_popular_queries_last_searched;
DROP INDEX IF EXISTS idx_search_popular_queries_count;
DROP INDEX IF EXISTS idx_search_zero_results_created;
DROP INDEX IF EXISTS idx_search_zero_results_query_hash;
DROP INDEX IF EXISTS idx_search_synonyms_category;
DROP INDEX IF EXISTS idx_search_synonyms_language;
DROP INDEX IF EXISTS idx_search_synonyms_term;

-- Drop tables
DROP TABLE IF EXISTS search_popular_queries;
DROP TABLE IF EXISTS search_zero_results;
DROP TABLE IF EXISTS search_synonyms;

-- Drop trigram indexes
DROP INDEX IF EXISTS idx_listings_description_trgm;
DROP INDEX IF EXISTS idx_listings_title_trgm;

-- Drop pg_trgm extension
DROP EXTENSION IF EXISTS pg_trgm;
