-- =============================================================================
-- Migration: 071_add_local_knowledge_vector.up.sql
-- Sprint: Stealth Mode Local Monopoly (RAG Architecture)
-- Purpose: Enable pgvector and create the local_knowledge table for
--          hyper-local community knowledge (hidden gems, cafés, routes, etc.)
--          to power Retrieval-Augmented Generation (RAG) on Miru AI.
-- Date: 2026-02-24
-- =============================================================================

-- 1. Enable the pgvector extension (idempotent)
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Create the local_knowledge table
CREATE TABLE IF NOT EXISTS local_knowledge (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    city        VARCHAR(100) NOT NULL,           -- e.g. 'Bandung', 'Bali', 'Yogyakarta'
    name        VARCHAR(255) NOT NULL,           -- e.g. 'Warung Nasi Ibu Entin'
    description TEXT        NOT NULL,           -- Rich text description for the RAG context
    category    VARCHAR(50)  NOT NULL,           -- e.g. 'cafe', 'route', 'attraction', 'restaurant'
    embedding   vector(1536),                   -- OpenAI text-embedding-3-small dimension (nullable until computed)
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- 3. Create HNSW index on embedding column using cosine distance
--    HNSW (Hierarchical Navigable Small World) provides lightning-fast approximate nearest-neighbor search.
--    vector_cosine_ops is optimal for normalized embeddings from OpenAI models.
CREATE INDEX IF NOT EXISTS idx_local_knowledge_embedding_hnsw
    ON local_knowledge
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- 4. Add a standard B-tree index on city for efficient pre-filtering
CREATE INDEX IF NOT EXISTS idx_local_knowledge_city
    ON local_knowledge (city);

-- 5. Add a B-tree index on category for efficient filtering
CREATE INDEX IF NOT EXISTS idx_local_knowledge_category
    ON local_knowledge (category);
