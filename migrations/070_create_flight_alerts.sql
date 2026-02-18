-- Migration: Create Flight Price Alerts (Flight Guardian)
-- Date: 2026-02-17
-- Sprint: 09 - Flight Guardian

-- =====================================================
-- TABLE: flight_price_alerts
-- Purpose: Track flight prices for existing trips
-- Philosophy: ONE alert per trip per route
-- =====================================================

CREATE TABLE flight_price_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE, -- Changed to TEXT to match trips.id
    origin_airport VARCHAR(3) NOT NULL, -- IATA code (e.g., "CGK")
    destination_airport VARCHAR(3) NOT NULL, -- IATA code (e.g., "NRT")
    departure_date DATE NOT NULL, -- From trip start date
    return_date DATE, -- From trip end date (optional for one-way)
    initial_price DECIMAL(10, 2) NOT NULL, -- Baseline when tracking started
    current_price DECIMAL(10, 2) NOT NULL, -- Latest fetched price
    lowest_price_seen DECIMAL(10, 2) NOT NULL, -- Record low
    currency VARCHAR(3) DEFAULT 'USD' NOT NULL,
    last_checked_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Prevent duplicate monitors for same route on same trip
CREATE UNIQUE INDEX idx_flight_alerts_unique_trip_route 
ON flight_price_alerts(trip_id, origin_airport);

-- Fast lookup by trip_id
CREATE INDEX idx_flight_alerts_trip_id ON flight_price_alerts(trip_id);

-- Fast lookup for active alerts (for background worker)
CREATE INDEX idx_flight_alerts_active ON flight_price_alerts(is_active) 
WHERE is_active = true;

-- =====================================================
-- COMMENTS
-- =====================================================

COMMENT ON TABLE flight_price_alerts IS 'Flight Guardian: Passive price monitoring for user trips';
COMMENT ON COLUMN flight_price_alerts.trip_id IS 'FK to trips table - one alert per trip per route';
COMMENT ON COLUMN flight_price_alerts.initial_price IS 'Baseline price when user activated Guardian';
COMMENT ON COLUMN flight_price_alerts.lowest_price_seen IS 'Best price ever seen for this route';
COMMENT ON INDEX idx_flight_alerts_unique_trip_route IS 'Prevent duplicate tracking for same route';
