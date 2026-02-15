-- Migration: Create place_library for enrichment caching
-- 051_create_place_library.sql

CREATE TABLE IF NOT EXISTS place_library (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    google_place_id TEXT,
    description TEXT,
    photos JSONB, -- Stores array of photo URLs or objects
    rating DOUBLE PRECISION,
    category TEXT,
    address TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    website TEXT,
    phone TEXT,
    opening_hours JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index on name for fast lookups and to prevent duplicates
-- Use LOWER(name) for case-insensitive matching during lookup
CREATE UNIQUE INDEX IF NOT EXISTS idx_place_library_name ON place_library (LOWER(name));
CREATE INDEX IF NOT EXISTS idx_place_library_google_id ON place_library (google_place_id);
