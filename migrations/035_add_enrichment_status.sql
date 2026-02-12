-- Add enrichment_status to trips table
ALTER TABLE trips ADD COLUMN enrichment_status VARCHAR(50) DEFAULT 'pending';
