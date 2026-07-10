-- Fix: trips.style VARCHAR(50) → TEXT
-- The style field is a generated string combining pace, social vibe, and budget preferences.
-- In practice it exceeds 50 chars (e.g. "Balanced mix of rest and activity, Mix of popular spots and local secrets, Mid-range comfort, good value").
ALTER TABLE trips ALTER COLUMN style TYPE TEXT;
