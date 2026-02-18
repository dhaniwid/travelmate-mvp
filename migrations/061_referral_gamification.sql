-- Migration: 061_referral_gamification.sql
-- Description: Add gamification features to referral system (leaderboard, achievements, milestones)
-- Date: 2026-02-17

-- ============================================================
-- 1. Add achievements tracking to users table
-- ============================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS achievements_unlocked JSONB DEFAULT '[]'::jsonb;

COMMENT ON COLUMN users.achievements_unlocked IS 'Array of unlocked achievement objects with tier, unlocked_at timestamp';

-- ============================================================
-- 2. Create leaderboard materialized view for performance
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS referral_leaderboard AS
SELECT 
    u.user_id,
    u.name,
    u.email,
    u.referral_code,
    COUNT(r.id) as total_referrals,
    u.bonus_trip_quota,
    u.achievements_unlocked,
    ROW_NUMBER() OVER (ORDER BY COUNT(r.id) DESC, MIN(r.created_at) ASC) as rank,
    MIN(r.created_at) as first_referral_at,
    MAX(r.created_at) as latest_referral_at
FROM users u
LEFT JOIN referrals r ON u.user_id = r.referrer_id
WHERE r.status = 'completed'
GROUP BY u.user_id, u.name, u.email, u.referral_code, u.bonus_trip_quota, u.achievements_unlocked
HAVING COUNT(r.id) > 0
ORDER BY total_referrals DESC, first_referral_at ASC
LIMIT 100;

-- ============================================================
-- 3. Create indexes for leaderboard performance
-- ============================================================
CREATE UNIQUE INDEX IF NOT EXISTS idx_leaderboard_user ON referral_leaderboard(user_id);
CREATE INDEX IF NOT EXISTS idx_leaderboard_rank ON referral_leaderboard(rank);
CREATE INDEX IF NOT EXISTS idx_leaderboard_referrals ON referral_leaderboard(total_referrals DESC);

-- ============================================================
-- 4. Create function to refresh leaderboard
-- ============================================================
CREATE OR REPLACE FUNCTION refresh_referral_leaderboard()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY referral_leaderboard;
    RAISE NOTICE 'Referral leaderboard refreshed at %', NOW();
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_referral_leaderboard IS 'Refresh leaderboard materialized view (can be called via cron job)';

-- ============================================================
-- 5. Create milestone tier definitions (reference data)
-- ============================================================
-- This is stored as a comment for reference, actual milestone logic is in Go service
-- 
-- Milestone Tiers:
-- - BRONZE:    1 referral   → Badge + Already granted in base system
-- - SILVER:    5 referrals  → Badge + 3 bonus trips
-- - GOLD:     10 referrals  → Badge + 1 week PRO trial (7 days)
-- - PLATINUM: 25 referrals  → Badge + 1 month PRO upgrade (30 days)
-- - DIAMOND:  50 referrals  → Badge + 3 months PRO upgrade (90 days)

-- ============================================================
-- 6. Index for faster achievement queries
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_users_achievements ON users USING GIN (achievements_unlocked);

-- ============================================================
-- 7. Trigger to auto-refresh leaderboard on referral insert
-- ============================================================
-- Note: For production, consider using a background job instead
-- to avoid blocking on large datasets

CREATE OR REPLACE FUNCTION trigger_refresh_leaderboard()
RETURNS TRIGGER AS $$
BEGIN
    -- Refresh asynchronously in production
    -- For now, we'll do it synchronously
    PERFORM refresh_referral_leaderboard();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Only refresh on completed referrals
CREATE TRIGGER after_referral_completed
AFTER INSERT OR UPDATE OF status ON referrals
FOR EACH ROW
WHEN (NEW.status = 'completed')
EXECUTE FUNCTION trigger_refresh_leaderboard();

-- ============================================================
-- 8. Verification queries
-- ============================================================

-- Check if achievements column was added
-- SELECT column_name, data_type FROM information_schema.columns 
-- WHERE table_name = 'users' AND column_name = 'achievements_unlocked';

-- View leaderboard
-- SELECT * FROM referral_leaderboard LIMIT 10;

-- Manually refresh leaderboard
-- SELECT refresh_referral_leaderboard();
