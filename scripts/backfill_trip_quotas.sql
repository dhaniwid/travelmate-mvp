-- Backfill script for trip_quotas based on existing trips
-- Run this once to populate quotas for users who already have trips.

INSERT INTO trip_quotas (user_id, month, trips_created, quota_limit, last_reset)
SELECT 
    user_id, 
    to_char(CURRENT_DATE, 'YYYY-MM') as month,
    COUNT(*) as trips_created,
    3 as quota_limit, -- Default for Free tier
    NOW() as last_reset
FROM trips
WHERE user_id IS NOT NULL AND user_id != 'guest'
GROUP BY user_id
ON CONFLICT (user_id) DO UPDATE SET
    trips_created = EXCLUDED.trips_created,
    updated_at = NOW();
