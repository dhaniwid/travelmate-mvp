-- Migration: Add ai_edits_used to trips table
-- 052_add_ai_edits_used.sql

ALTER TABLE trips ADD COLUMN IF NOT EXISTS ai_edits_used INTEGER DEFAULT 0;
