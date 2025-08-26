-- PostgreSQL schema for developer blog indexing
-- Designed for efficient full-text search and content deduplication

-- Main table for indexed blog posts
CREATE TABLE IF NOT EXISTS pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url VARCHAR(2048) UNIQUE NOT NULL,
    url_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA-256 hash for deduplication
    
    -- Content metadata
    title TEXT,
    description TEXT,
    author VARCHAR(512),
    
    -- Searchable content
    headings TEXT[], -- Array of H1, H2, H3 tags
    content TEXT, -- Concatenated paragraphs for full-text search
    keywords TEXT[], -- Extracted keywords
    
    -- Technical metadata
    links TEXT[], -- Outbound links
    script_links TEXT[], -- JavaScript references
    domain VARCHAR(255), -- Extracted from URL for filtering
    
    -- Content fingerprinting for deduplication
    content_hash VARCHAR(64), -- SHA-256 of normalized content
    word_count INTEGER, -- For content quality scoring
    
    -- Timestamps
    created_at BIGINT NOT NULL, -- Unix timestamp from crawler
    indexed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Full-text search vector (automatically maintained)
    search_vector tsvector
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pages_url_hash ON pages (url_hash);
CREATE INDEX IF NOT EXISTS idx_pages_content_hash ON pages (content_hash);
CREATE INDEX IF NOT EXISTS idx_pages_domain ON pages (domain);
CREATE INDEX IF NOT EXISTS idx_pages_created_at ON pages (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pages_word_count ON pages (word_count);

-- Full-text search index using GIN
CREATE INDEX IF NOT EXISTS idx_pages_search 
    ON pages USING GIN (search_vector);

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_pages_domain_created 
    ON pages (domain, created_at DESC);

-- Function to extract domain from URL
CREATE OR REPLACE FUNCTION extract_domain(url TEXT) 
RETURNS TEXT AS $$
BEGIN
    RETURN regexp_replace(
        regexp_replace(url, '^https?://', ''),
        '/.*$', ''
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to calculate word count
CREATE OR REPLACE FUNCTION word_count(text_content TEXT)
RETURNS INTEGER AS $$
BEGIN
    IF text_content IS NULL OR text_content = '' THEN
        RETURN 0;
    END IF;
    
    RETURN array_length(
        string_to_array(
            regexp_replace(text_content, '[^\w\s]', ' ', 'g'),
            ' '
        ), 
        1
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger function to auto-update search_vector, domain, and word_count
CREATE OR REPLACE FUNCTION update_page_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Update search vector with weighted content
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(array_to_string(NEW.headings, ' '), '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(array_to_string(NEW.keywords, ' '), '')), 'D');
    
    -- Extract domain
    NEW.domain := extract_domain(NEW.url);
    
    -- Calculate word count
    NEW.word_count := word_count(NEW.content);
    
    -- Update timestamp
    NEW.updated_at := NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trigger_update_search_vector ON pages;
CREATE TRIGGER trigger_update_search_vector
    BEFORE INSERT OR UPDATE ON pages
    FOR EACH ROW EXECUTE FUNCTION update_page_search_vector();

-- View for search results with ranking
CREATE OR REPLACE VIEW search_results AS
SELECT 
    id,
    url,
    title,
    description,
    author,
    domain,
    word_count,
    created_at,
    indexed_at,
    ts_rank_cd(search_vector, plainto_tsquery('english', '')) as rank
FROM pages;

-- Table for tracking processing statistics
CREATE TABLE IF NOT EXISTS indexer_stats (
    date DATE PRIMARY KEY DEFAULT CURRENT_DATE,
    pages_processed INTEGER DEFAULT 0,
    pages_inserted INTEGER DEFAULT 0,
    pages_updated INTEGER DEFAULT 0,
    pages_skipped INTEGER DEFAULT 0,
    processing_errors INTEGER DEFAULT 0,
    avg_processing_time_ms NUMERIC(10,2),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Function to update daily stats
CREATE OR REPLACE FUNCTION update_indexer_stats(
    p_processed INTEGER DEFAULT 1,
    p_inserted INTEGER DEFAULT 0,
    p_updated INTEGER DEFAULT 0,
    p_skipped INTEGER DEFAULT 0,
    p_errors INTEGER DEFAULT 0,
    p_processing_time_ms NUMERIC DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO indexer_stats (
        date, pages_processed, pages_inserted, pages_updated, 
        pages_skipped, processing_errors, avg_processing_time_ms
    )
    VALUES (
        CURRENT_DATE, p_processed, p_inserted, p_updated, 
        p_skipped, p_errors, p_processing_time_ms
    )
    ON CONFLICT (date) DO UPDATE SET
        pages_processed = indexer_stats.pages_processed + p_processed,
        pages_inserted = indexer_stats.pages_inserted + p_inserted,
        pages_updated = indexer_stats.pages_updated + p_updated,
        pages_skipped = indexer_stats.pages_skipped + p_skipped,
        processing_errors = indexer_stats.processing_errors + p_errors,
        avg_processing_time_ms = CASE 
            WHEN p_processing_time_ms IS NOT NULL THEN
                (COALESCE(indexer_stats.avg_processing_time_ms, 0) + p_processing_time_ms) / 2
            ELSE indexer_stats.avg_processing_time_ms
        END,
        last_updated = NOW();
END;
$$ LANGUAGE plpgsql;