CREATE TABLE IF NOT EXISTS route_prices
(
    id           TEXT PRIMARY KEY,
    route_id     TEXT REFERENCES routes (id) ON DELETE CASCADE, -- Link ke tabel Routes
    provider     VARCHAR(100),
    price_amount BIGINT,
    travel_date  VARCHAR(20),                                   -- YYYY-MM-DD
    recorded_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index agar query history cepat
CREATE INDEX idx_route_prices_route_id ON route_prices (route_id);