-- Drop dependent tables first to avoid foreign key constraints
DROP TABLE IF EXISTS user_interest_logs;
DROP TABLE IF EXISTS destination_vibes;
DROP TABLE IF EXISTS destinations;

-- Create the new destinations table with UUID
CREATE TABLE destinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    country VARCHAR(255) NOT NULL,
    description TEXT,
    image_url TEXT NOT NULL,
    category VARCHAR(50), -- 'Nature', 'City', 'Beach', 'Culinary'
    tags JSONB,           -- ["visa-free", "spring", "trending"]
    is_trending BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Keep this for compatibility if other parts of the system rely on it, 
    -- but mapped to the new structure
    popularity_score INT DEFAULT 0,
    discovery_data JSONB -- Keeping this column as it was present in the old schema/repo
);

-- Index for performance
CREATE INDEX idx_destinations_trending ON destinations(is_trending);
CREATE INDEX idx_destinations_category ON destinations(category);
