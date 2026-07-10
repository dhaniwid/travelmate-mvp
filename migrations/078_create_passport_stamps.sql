-- MT-44: Digital Passport — create passport_stamps table
CREATE TABLE IF NOT EXISTS passport_stamps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(255) NOT NULL,
    city        VARCHAR(100) NOT NULL,
    city_slug   VARCHAR(100) NOT NULL,
    date        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    serial      VARCHAR(20) NOT NULL,
    mood        VARCHAR(10) NOT NULL CHECK (mood IN ('morning', 'rain', 'night')),
    image_url   VARCHAR(255) NOT NULL,
    rotation    DECIMAL(4,2) NOT NULL,
    trip_id     TEXT REFERENCES trips(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, city_slug, mood)
);

CREATE INDEX IF NOT EXISTS idx_passport_stamps_user_id ON passport_stamps(user_id);
CREATE INDEX IF NOT EXISTS idx_passport_stamps_trip_id ON passport_stamps(trip_id);
