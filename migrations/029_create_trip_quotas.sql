-- Migration: 029_create_trip_quotas
-- Description: Create table to track monthly trip creation quotas
-- Author: Engineering Team
-- Date: 2026-02-11

-- UP MIGRATION
BEGIN;

DROP TABLE IF EXISTS trip_quotas CASCADE;

-- Create trip_quotas table
CREATE TABLE trip_quotas (
    user_id VARCHAR(255) PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    month VARCHAR(7) NOT NULL,           -- Format: "YYYY-MM" (e.g., "2026-02")
    trips_created INTEGER DEFAULT 0 NOT NULL CHECK (trips_created >= 0),
    quota_limit INTEGER DEFAULT 3 NOT NULL CHECK (quota_limit > 0),
    last_reset TIMESTAMP DEFAULT NOW() NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

-- Add indexes
CREATE INDEX idx_quotas_month ON trip_quotas(month);
CREATE INDEX idx_quotas_user_month ON trip_quotas(user_id, month);

-- Create trigger for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_trip_quotas_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trip_quotas_updated_at_trigger
BEFORE UPDATE ON trip_quotas
FOR EACH ROW
EXECUTE FUNCTION update_trip_quotas_timestamp();

COMMIT;
