-- Migration: Create Gold Schema Tables (Unified)
-- Run this on ola-b2b database (hana_securities)

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS gold;

-- Unified Translated News Table
-- Combines JP Minkabu and CN Wind news into a single table
CREATE TABLE IF NOT EXISTS gold.translated_news (
    id                  BIGSERIAL PRIMARY KEY,
    
    -- Source identification
    source              VARCHAR(20) NOT NULL,      -- 'jp_minkabu' | 'cn_wind'
    source_news_id      VARCHAR(255) NOT NULL,     -- Original news_id or object_id from source
    
    -- Original content
    original_headline   TEXT NOT NULL,
    original_content    TEXT,
    
    -- Translated content
    translated_headline TEXT NOT NULL,
    translated_content  TEXT,
    
    -- Categorization & Search
    tickers             TEXT[] DEFAULT '{}',       -- Stock tickers (JP tickers or CN wind_codes)
    topics              TEXT[] DEFAULT '{}',       -- Topics/sections/categories
    keywords            TEXT[] DEFAULT '{}',       -- Keywords for search
    
    -- Metadata
    provider            VARCHAR(100),              -- News provider name
    published_at        TIMESTAMPTZ NOT NULL,      -- Original publish/creation time
    model_name          VARCHAR(100) NOT NULL,     -- Translation model used
    
    -- Sync tracking
    source_created_at   TIMESTAMPTZ,
    source_updated_at   TIMESTAMPTZ,
    synced_at           TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique constraint: one record per source + source_news_id
    UNIQUE(source, source_news_id)
);

-- Indexes for efficient querying
-- GIN index for ticker-based search (main use case)
CREATE INDEX IF NOT EXISTS idx_news_tickers_gin 
    ON gold.translated_news USING GIN (tickers);

-- GIN index for topics search
CREATE INDEX IF NOT EXISTS idx_news_topics_gin 
    ON gold.translated_news USING GIN (topics);

-- GIN index for keywords search  
CREATE INDEX IF NOT EXISTS idx_news_keywords_gin 
    ON gold.translated_news USING GIN (keywords);

-- B-tree indexes for filtering and sorting
CREATE INDEX IF NOT EXISTS idx_news_published_at 
    ON gold.translated_news (published_at DESC);

CREATE INDEX IF NOT EXISTS idx_news_source 
    ON gold.translated_news (source);

CREATE INDEX IF NOT EXISTS idx_news_source_published 
    ON gold.translated_news (source, published_at DESC);

CREATE INDEX IF NOT EXISTS idx_news_synced_at 
    ON gold.translated_news (synced_at);

-- Sync metadata table (tracks last sync time per source)
CREATE TABLE IF NOT EXISTS gold.sync_metadata (
    id              SERIAL PRIMARY KEY,
    source          VARCHAR(20) NOT NULL UNIQUE,  -- 'jp_minkabu' | 'cn_wind'
    last_synced_at  TIMESTAMPTZ,
    last_sync_count INT DEFAULT 0,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Initialize sync metadata
INSERT INTO gold.sync_metadata (source, last_synced_at) 
VALUES 
    ('jp_minkabu', NULL),
    ('cn_wind', NULL)
ON CONFLICT (source) DO NOTHING;

-- Comments
COMMENT ON TABLE gold.translated_news IS 'Unified translated news from multiple sources (JP Minkabu, CN Wind)';
COMMENT ON COLUMN gold.translated_news.source IS 'News source: jp_minkabu or cn_wind';
COMMENT ON COLUMN gold.translated_news.source_news_id IS 'Original unique ID from the source system';
COMMENT ON COLUMN gold.translated_news.tickers IS 'Stock ticker codes for search (JP tickers or CN wind_codes)';
COMMENT ON TABLE gold.sync_metadata IS 'Tracks ETL sync progress for each news source';
