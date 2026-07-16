-- MT-53: Miru Radar — add lat/lng + landmark columns to local_knowledge
ALTER TABLE local_knowledge ADD COLUMN IF NOT EXISTS lat DECIMAL(9,6);
ALTER TABLE local_knowledge ADD COLUMN IF NOT EXISTS lng DECIMAL(9,6);
ALTER TABLE local_knowledge ADD COLUMN IF NOT EXISTS has_landmark_svg BOOLEAN DEFAULT FALSE;
ALTER TABLE local_knowledge ADD COLUMN IF NOT EXISTS landmark_slug VARCHAR(100);
ALTER TABLE local_knowledge ADD COLUMN IF NOT EXISTS stamp_radius_meters INTEGER DEFAULT 500;

-- GIST index on built-in point type for proximity queries
CREATE INDEX IF NOT EXISTS idx_local_knowledge_location
    ON local_knowledge USING GIST (point(lng, lat))
    WHERE lat IS NOT NULL AND lng IS NOT NULL;
