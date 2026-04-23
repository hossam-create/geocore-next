-- ════════════════════════════════════════════════════════════════════════════
-- 054: Search Quality Improvements (pg_trgm, synonyms, analytics, ranking)
-- ════════════════════════════════════════════════════════════════════════════

-- 1. Enable pg_trgm extension for typo tolerance and similarity search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. Add GIN trigram indexes on listings for fast similarity search
-- These enable % (similarity) and <-> (distance) operators
CREATE INDEX IF NOT EXISTS idx_listings_title_trgm 
    ON listings USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_listings_description_trgm 
    ON listings USING gin (description gin_trgm_ops);

-- 3. Search synonyms table for query expansion
CREATE TABLE IF NOT EXISTS search_synonyms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term            TEXT NOT NULL,           -- e.g., "phone"
    synonym         TEXT NOT NULL,           -- e.g., "mobile", "smartphone"
    language        TEXT NOT NULL DEFAULT 'en', -- en, ar, etc.
    category        TEXT,                    -- optional category scope
    weight          FLOAT NOT NULL DEFAULT 1.0, -- synonym relevance weight
    is_bidirectional BOOLEAN NOT NULL DEFAULT FALSE, -- if true, also maps synonym -> term
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_synonyms_term ON search_synonyms(term);
CREATE INDEX IF NOT EXISTS idx_search_synonyms_language ON search_synonyms(language);
CREATE INDEX IF NOT EXISTS idx_search_synonyms_category ON search_synonyms(category);

-- 4. Search analytics tables
-- Track zero-result queries for optimization
CREATE TABLE IF NOT EXISTS search_zero_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query           TEXT NOT NULL,
    query_hash      TEXT NOT NULL,           -- SHA256 for deduplication
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    intent_json     JSONB,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_zero_results_query_hash ON search_zero_results(query_hash);
CREATE INDEX IF NOT EXISTS idx_search_zero_results_created ON search_zero_results(created_at DESC);

-- Track popular queries for trending and optimization
CREATE TABLE IF NOT EXISTS search_popular_queries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query           TEXT NOT NULL,
    query_hash      TEXT NOT NULL UNIQUE,     -- deduplication
    search_count    INT NOT NULL DEFAULT 1,
    result_count_avg FLOAT,                  -- avg results per search
    last_searched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    language        TEXT NOT NULL DEFAULT 'en',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_popular_queries_count ON search_popular_queries(search_count DESC);
CREATE INDEX IF NOT EXISTS idx_search_popular_queries_last_searched ON search_popular_queries(last_searched_at DESC);

-- 5. Add columns to listings for ranking signals
ALTER TABLE listings 
    ADD COLUMN IF NOT EXISTS view_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS search_click_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS search_impression_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_searched_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_viewed_at TIMESTAMPTZ;

-- 5.1 Create listing_views table for view tracking
CREATE TABLE IF NOT EXISTS listing_views (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(listing_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_listing_views_listing ON listing_views(listing_id);
CREATE INDEX IF NOT EXISTS idx_listing_views_user ON listing_views(user_id);

-- 5.2 Create listing_clicks table for click tracking
CREATE TABLE IF NOT EXISTS listing_clicks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    clicked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(listing_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_listing_clicks_listing ON listing_clicks(listing_id);
CREATE INDEX IF NOT EXISTS idx_listing_clicks_user ON listing_clicks(user_id);

-- Indexes for ranking queries
CREATE INDEX IF NOT EXISTS idx_listings_view_count ON listings(view_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_search_click_count ON listings(search_click_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_last_searched ON listings(last_searched_at DESC);
CREATE INDEX IF NOT EXISTS idx_listings_last_viewed ON listings(last_viewed_at DESC);

-- 6. Update search_queries table with additional analytics
ALTER TABLE search_queries 
    ADD COLUMN IF NOT EXISTS query_hash TEXT,
    ADD COLUMN IF NOT EXISTS zero_result BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS clicked_listing_id UUID REFERENCES listings(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS position_in_results INT,     -- which position user clicked
    ADD COLUMN IF NOT EXISTS filters_json JSONB;

CREATE INDEX IF NOT EXISTS idx_search_queries_query_hash ON search_queries(query_hash);
CREATE INDEX IF NOT EXISTS idx_search_queries_zero_result ON search_queries(zero_result) WHERE zero_result = TRUE;
CREATE INDEX IF NOT EXISTS idx_search_queries_clicked ON search_queries(clicked_listing_id);

-- 7. Seed initial synonyms (English)
INSERT INTO search_synonyms (term, synonym, language, weight, is_bidirectional) VALUES
    ('phone', 'mobile', 'en', 1.0, true),
    ('phone', 'smartphone', 'en', 1.0, true),
    ('phone', 'cellphone', 'en', 0.9, true),
    ('car', 'vehicle', 'en', 1.0, true),
    ('car', 'automobile', 'en', 0.8, true),
    ('apartment', 'flat', 'en', 1.0, true),
    ('apartment', 'condo', 'en', 0.9, true),
    ('house', 'home', 'en', 1.0, true),
    ('laptop', 'notebook', 'en', 1.0, true),
    ('watch', 'timepiece', 'en', 0.8, true),
    ('sneakers', 'shoes', 'en', 1.0, true),
    ('sneakers', 'trainers', 'en', 0.9, true)
ON CONFLICT DO NOTHING;

-- 8. Seed initial synonyms (Arabic)
INSERT INTO search_synonyms (term, synonym, language, weight, is_bidirectional) VALUES
    ('هاتف', 'موبايل', 'ar', 1.0, true),
    ('هاتف', 'جوال', 'ar', 1.0, true),
    ('سيارة', 'مركبة', 'ar', 1.0, true),
    ('شقة', 'وحدة سكنية', 'ar', 0.9, true),
    ('شقة', 'شقة سكنية', 'ar', 1.0, true),
    ('منزل', 'بيت', 'ar', 1.0, true),
    ('لابتوب', 'حاسوب محمول', 'ar', 1.0, true),
    ('ساعة', 'ساعة يد', 'ar', 0.9, true),
    ('حذاء', 'حذاء رياضي', 'ar', 1.0, true)
ON CONFLICT DO NOTHING;

-- 9. Create function to update popular queries incrementally
CREATE OR REPLACE FUNCTION update_popular_query(
    p_query TEXT,
    p_query_hash TEXT,
    p_result_count INT,
    p_language TEXT DEFAULT 'en'
) RETURNS VOID AS $$
BEGIN
    INSERT INTO search_popular_queries (query, query_hash, search_count, result_count_avg, last_searched_at, first_seen_at, language)
    VALUES (p_query, p_query_hash, 1, p_result_count::FLOAT, NOW(), NOW(), p_language)
    ON CONFLICT (query_hash) 
    DO UPDATE SET
        search_count = search_popular_queries.search_count + 1,
        result_count_avg = (search_popular_queries.result_count_avg * search_popular_queries.search_count + p_result_count::FLOAT) / (search_popular_queries.search_count + 1),
        last_searched_at = NOW(),
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- 10. Create function to expand query with synonyms
CREATE OR REPLACE FUNCTION expand_query_with_synonyms(
    p_query TEXT,
    p_language TEXT DEFAULT 'en'
) RETURNS TEXT[] AS $$
DECLARE
    expanded_terms TEXT[] := ARRAY[p_query];
    synonyms TEXT[];
BEGIN
    -- Find synonyms for each word in the query
    SELECT array_agg(DISTINCT synonym) INTO synonyms
    FROM search_synonyms
    WHERE term = ANY(string_to_array(lower(p_query), ' '))
    AND language = p_language;
    
    IF synonyms IS NOT NULL THEN
        expanded_terms := array_cat(expanded_terms, synonyms);
    END IF;
    
    RETURN expanded_terms;
END;
$$ LANGUAGE plpgsql;

-- Documentation
COMMENT ON EXTENSION pg_trgm IS 'Provides trigram matching for fuzzy string matching and similarity search';
COMMENT ON TABLE search_synonyms IS 'Query expansion synonyms for improved search recall';
COMMENT ON TABLE search_zero_results IS 'Tracks queries that returned zero results for optimization';
COMMENT ON TABLE search_popular_queries IS 'Tracks popular search queries for trending and analytics';
COMMENT ON FUNCTION update_popular_query IS 'Incrementally updates popular query statistics';
COMMENT ON FUNCTION expand_query_with_synonyms IS 'Expands search query with synonyms for better recall';
