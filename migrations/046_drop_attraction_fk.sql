-- Migration 046: Drop Foreign Key constraint on tourist_attractions.location_id
-- We want this to be a global cache, not restricted to our featured 'locations' table.

ALTER TABLE tourist_attractions DROP CONSTRAINT IF EXISTS tourist_attractions_location_id_fkey;
