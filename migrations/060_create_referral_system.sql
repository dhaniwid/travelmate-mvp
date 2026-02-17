-- Migration: 060_create_referral_system.sql
-- Purpose: Add referral system to enable viral growth
-- Users earn +1 trip quota for each successful referral

-- ==========================================
-- 1. UPDATE USERS TABLE
-- ==========================================

-- Add referral code column (unique identifier for sharing)
ALTER TABLE users ADD COLUMN referral_code VARCHAR(20) UNIQUE;

-- Add bonus trip quota column (tracks earned rewards)
ALTER TABLE users ADD COLUMN bonus_trip_quota INT DEFAULT 0 NOT NULL;

-- ==========================================
-- 2. CREATE REFERRALS TABLE
-- ==========================================

CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    referred_user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'completed' NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    
    -- Constraints
    CONSTRAINT no_self_referral CHECK (referrer_id != referred_user_id),
    CONSTRAINT unique_referral UNIQUE(referred_user_id)
);

-- ==========================================
-- 3. INDEXES FOR PERFORMANCE
-- ==========================================

-- Index for fetching user's referral stats
CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);

-- Index for checking if user was already referred
CREATE INDEX idx_referrals_referred_user ON referrals(referred_user_id);

-- ==========================================
-- 4. BACKFILL REFERRAL CODES FOR EXISTING USERS
-- ==========================================

-- Function to generate unique referral code
CREATE OR REPLACE FUNCTION generate_referral_code() RETURNS VARCHAR(20) AS $$
DECLARE
    chars TEXT := 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'; -- Exclude confusing chars (0, O, I, 1)
    code TEXT := 'MIRU-';
    i INTEGER;
BEGIN
    FOR i IN 1..4 LOOP
        code := code || substr(chars, floor(random() * length(chars) + 1)::int, 1);
    END LOOP;
    RETURN code;
END;
$$ LANGUAGE plpgsql;

-- Backfill codes for existing users
DO $$
DECLARE
    user_record RECORD;
    new_code VARCHAR(20);
    max_attempts INT := 10;
    attempt INT;
BEGIN
    FOR user_record IN SELECT user_id FROM users WHERE referral_code IS NULL LOOP
        attempt := 0;
        LOOP
            new_code := generate_referral_code();
            
            -- Try to update with unique code
            BEGIN
                UPDATE users 
                SET referral_code = new_code 
                WHERE user_id = user_record.user_id;
                EXIT; -- Success, exit loop
            EXCEPTION WHEN unique_violation THEN
                attempt := attempt + 1;
                IF attempt >= max_attempts THEN
                    RAISE EXCEPTION 'Failed to generate unique code for user %', user_record.user_id;
                END IF;
            END;
        END LOOP;
    END LOOP;
END $$;

-- Make referral_code NOT NULL after backfill
ALTER TABLE users ALTER COLUMN referral_code SET NOT NULL;

-- ==========================================
-- 5. COMMENTS FOR DOCUMENTATION
-- ==========================================

COMMENT ON TABLE referrals IS 'Tracks referral relationships and rewards';
COMMENT ON COLUMN users.referral_code IS 'Unique shareable code for inviting friends (format: MIRU-XXXX)';
COMMENT ON COLUMN users.bonus_trip_quota IS 'Additional trip quota earned through referrals';
COMMENT ON CONSTRAINT no_self_referral ON referrals IS 'Prevents users from referring themselves';
COMMENT ON CONSTRAINT unique_referral ON referrals IS 'Ensures each user can only be referred once';
