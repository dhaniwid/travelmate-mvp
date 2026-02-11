-- Migration: 028_add_subscription_to_users (Replacing broken schema)
-- Description: Recreate users table with correct Clerk ID support
-- Author: Engineering Team
-- Date: 2026-02-11

-- UP MIGRATION
BEGIN;

-- Drop dependent triggers first if they exist (from failed attempts)
DROP TRIGGER IF EXISTS users_subscription_change_trigger ON users;
DROP FUNCTION IF EXISTS log_subscription_change();

-- Drop old users table (and dependent FKs like user_interest_logs)
DROP TABLE IF EXISTS users CASCADE;

-- Recreate users table with correct schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,                        -- Internal DB ID
    user_id VARCHAR(255) UNIQUE NOT NULL,         -- External Auth ID (Clerk)
    email VARCHAR(255),
    name VARCHAR(255),
    
    -- Subscription status
    subscription_tier VARCHAR(20) DEFAULT 'FREE' CHECK (subscription_tier IN ('FREE', 'PRO')),
    subscription_status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (subscription_status IN ('ACTIVE', 'CANCELLED', 'EXPIRED')),
    subscription_started_at TIMESTAMP,
    subscription_ends_at TIMESTAMP,
    stripe_customer_id VARCHAR(255) UNIQUE,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_user_id ON users(user_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription_tier ON users(subscription_tier);
CREATE INDEX idx_users_stripe_customer ON users(stripe_customer_id);

-- Comments
COMMENT ON COLUMN users.user_id IS 'External Auth ID (Clerk Subject)';
COMMENT ON COLUMN users.subscription_tier IS 'User subscription tier: FREE or PRO';

COMMIT;
