-- Migration 044: Add caching fields to tourist_attractions
ALTER TABLE tourist_attractions ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION;
ALTER TABLE tourist_attractions ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;
ALTER TABLE tourist_attractions ADD COLUMN IF NOT EXISTS place_id TEXT;
ALTER TABLE tourist_attractions ADD COLUMN IF NOT EXISTS photos JSONB DEFAULT '[]';
ALTER TABLE tourist_attractions ADD COLUMN IF NOT EXISTS visit_duration TEXT;

-- Index for searching by name and location (standardized)
-- Note: Existing index idx_attraction_name_location already exists on (name, location_id)
-- We add one specifically for place_id if it becomes our primary cache key
CREATE INDEX IF NOT EXISTS idx_attraction_place_id ON tourist_attractions (place_id);
