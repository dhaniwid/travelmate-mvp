-- Create trip_collaborators table for multi-user trip access
-- Sprint 7: Trip Collaboration

CREATE TABLE IF NOT EXISTS trip_collaborators (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE, -- Matched TEXT type for FK
    user_id VARCHAR(255) NOT NULL,  -- Clerk User ID
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',  -- 'owner', 'editor', 'viewer'
    status VARCHAR(20) NOT NULL DEFAULT 'accepted',  -- 'pending', 'accepted', 'declined'
    invited_by VARCHAR(255),  -- User ID who sent the invite
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure no duplicate collaborators per trip
    UNIQUE(trip_id, user_id)
);

-- Index for quickly finding trips shared with a user
CREATE INDEX IF NOT EXISTS idx_collaborators_user ON trip_collaborators(user_id);

-- Index for quickly finding collaborators of a trip
CREATE INDEX IF NOT EXISTS idx_collaborators_trip ON trip_collaborators(trip_id);

-- Index for filtering by status
CREATE INDEX IF NOT EXISTS idx_collaborators_status ON trip_collaborators(status);

-- Backfill existing trips: Add current trip owners as 'owner' collaborators
-- This ensures backward compatibility with existing data
INSERT INTO trip_collaborators (trip_id, user_id, role, status, invited_by)
SELECT 
    id as trip_id,
    user_id,
    'owner' as role,
    'accepted' as status,
    user_id as invited_by
FROM trips
WHERE NOT EXISTS (
    SELECT 1 FROM trip_collaborators tc 
    WHERE tc.trip_id = trips.id AND tc.user_id = trips.user_id
);

-- Add comment for documentation
COMMENT ON TABLE trip_collaborators IS 'Stores trip collaboration relationships for multi-user access';
COMMENT ON COLUMN trip_collaborators.role IS 'Access level: owner (full control), editor (can modify), viewer (read-only)';
COMMENT ON COLUMN trip_collaborators.status IS 'Invitation status: pending (not yet accepted), accepted (active), declined (rejected invite)';
