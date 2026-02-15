-- Create user_analytics_events table for tracking user behavior and conversion metrics

CREATE TABLE IF NOT EXISTS user_analytics_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id VARCHAR(255) NOT NULL,
  event_type VARCHAR(50) NOT NULL,     -- e.g., "trip_created", "paywall_shown", "upgrade_clicked"
  event_data JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT NOW()
);

-- Optimize for user-based and type-based reporting
CREATE INDEX IF NOT EXISTS idx_analytics_user ON user_analytics_events(user_id);
CREATE INDEX IF NOT EXISTS idx_analytics_type ON user_analytics_events(event_type);
