-- Fix trips.user_id type: uuid → text
-- Clerk user IDs are strings (e.g. "user_2Nxyz..."), not UUIDs.
-- The uuid type causes "operator does not exist: character varying = uuid" errors.
ALTER TABLE trips ALTER COLUMN user_id TYPE TEXT USING user_id::text;
