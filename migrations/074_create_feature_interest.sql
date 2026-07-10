CREATE TABLE IF NOT EXISTS feature_interest (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id TEXT NOT NULL,
    feature_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_feature_interest_unique ON feature_interest(user_id, feature_key);
CREATE INDEX IF NOT EXISTS idx_feature_interest_feature ON feature_interest(feature_key);
