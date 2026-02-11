-- Migration: 030_create_subscription_events
-- Description: Create audit log for subscription lifecycle events
-- Author: Engineering Team
-- Date: 2026-02-11

-- UP MIGRATION
BEGIN;

DROP TABLE IF EXISTS subscription_events CASCADE;
DROP TRIGGER IF EXISTS users_subscription_change_trigger ON users;
DROP FUNCTION IF EXISTS log_subscription_change();

-- Create subscription_events table
CREATE TABLE subscription_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,     -- "upgraded", "downgraded", "cancelled", "renewed"
    from_tier VARCHAR(20),               -- Previous tier (nullable for first-time events)
    to_tier VARCHAR(20),                 -- New tier
    stripe_event_id VARCHAR(255),        -- For idempotency checking
    metadata JSONB DEFAULT '{}',         -- Additional event data
    created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

-- Add indexes for common queries
CREATE INDEX idx_sub_events_user_id ON subscription_events(user_id);
CREATE INDEX idx_sub_events_event_type ON subscription_events(event_type);
CREATE INDEX idx_sub_events_stripe_id ON subscription_events(stripe_event_id);
CREATE INDEX idx_sub_events_created_at ON subscription_events(created_at DESC);

-- Create function to auto-log subscription changes
CREATE OR REPLACE FUNCTION log_subscription_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if subscription_tier changed
    IF OLD.subscription_tier IS DISTINCT FROM NEW.subscription_tier THEN
        INSERT INTO subscription_events (user_id, event_type, from_tier, to_tier, metadata)
        VALUES (
            NEW.user_id,
            CASE
                WHEN NEW.subscription_tier = 'PRO' THEN 'upgraded'
                ELSE 'downgraded'
            END,
            OLD.subscription_tier,
            NEW.subscription_tier,
            jsonb_build_object(
                'old_status', OLD.subscription_status,
                'new_status', NEW.subscription_status,
                'stripe_customer_id', NEW.stripe_customer_id
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger on users table
CREATE TRIGGER users_subscription_change_trigger
AFTER UPDATE ON users
FOR EACH ROW
WHEN (OLD.subscription_tier IS DISTINCT FROM NEW.subscription_tier)
EXECUTE FUNCTION log_subscription_change();

COMMIT;
