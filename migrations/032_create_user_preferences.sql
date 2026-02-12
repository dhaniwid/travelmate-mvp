CREATE TABLE IF NOT EXISTS user_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    pace VARCHAR(50) DEFAULT 'BALANCED',   -- RELAXED, BALANCED, FAST
    budget_tier VARCHAR(50) DEFAULT 'MID', -- BUDGET, MID, LUXURY
    dietary JSONB DEFAULT '[]',             -- ["Vegetarian", "Halal"]
    interests JSONB DEFAULT '[]',            -- ["Culture", "Nature", "Adventure"]
    travel_style JSONB DEFAULT '[]',         -- ["Family", "Solo", "Romantic"]
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);
