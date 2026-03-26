-- Migration: Add pgvector extension and listing_embeddings table
  -- Run: psql $DATABASE_URL -f migrations/20260324_001_pgvector_search.sql

  -- Enable pgvector extension
  CREATE EXTENSION IF NOT EXISTS vector;

  -- Listing embeddings table for semantic search
  CREATE TABLE IF NOT EXISTS listing_embeddings (
      id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      listing_id   UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
      embedding    vector(1536),  -- OpenAI text-embedding-3-small dimension
      content_hash TEXT NOT NULL, -- SHA256 of title+description to detect staleness
      model        TEXT NOT NULL DEFAULT 'text-embedding-3-small',
      created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      UNIQUE(listing_id)
  );

  -- Index for fast cosine similarity search
  CREATE INDEX IF NOT EXISTS idx_listing_embeddings_vector
      ON listing_embeddings USING ivfflat (embedding vector_cosine_ops)
      WITH (lists = 100);

  -- Search queries audit log
  CREATE TABLE IF NOT EXISTS search_queries (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
      query       TEXT NOT NULL,
      intent_json JSONB,
      result_count INT,
      latency_ms  INT,
      created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );

  CREATE INDEX IF NOT EXISTS idx_search_queries_created ON search_queries(created_at DESC);
  