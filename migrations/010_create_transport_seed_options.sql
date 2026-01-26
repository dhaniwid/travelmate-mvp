CREATE TABLE IF NOT EXISTS transport_seed_options
(
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    origin_city         VARCHAR(100) NOT NULL,
    destination_city    VARCHAR(100) NOT NULL,
    transport_type      VARCHAR(50),  -- 'Flight', 'Train', 'Bus'
    provider_name       VARCHAR(100), -- 'Garuda Indonesia', 'Whoosh', 'CitiLink'
    estimated_price_min NUMERIC,
    estimated_duration  VARCHAR(50),
    pros                TEXT,         -- Alasan kenapa opsi ini bagus (dari AI)
    created_at          TIMESTAMP        DEFAULT NOW(),

    -- Constraint agar kita tidak menyimpan duplikat (Upsert logic base)
    CONSTRAINT unique_route_provider UNIQUE (origin_city, destination_city, provider_name, transport_type)
);