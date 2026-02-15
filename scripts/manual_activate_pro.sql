-- MANUAL PRO ACTIVATION SCRIPT
-- Purpose: Manually upgrade a user to PRO tier for 30 days.
-- Usage: Replace 'user@example.com' with the target user's email.

UPDATE users 
SET 
    subscription_tier = 'PRO',
    subscription_status = 'ACTIVE',
    subscription_started_at = NOW(),
    subscription_ends_at = NOW() + INTERVAL '30 days',
    updated_at = NOW()
WHERE id = 1;

-- Verification Query
SELECT email, subscription_tier, subscription_status, subscription_ends_at 
FROM users 
WHERE email = 'user@example.com';
