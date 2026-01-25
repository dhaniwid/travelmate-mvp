CREATE TABLE IF NOT EXISTS tourist_attractions
(
    id               UUID PRIMARY KEY,
    location_id      TEXT REFERENCES locations (id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    category         VARCHAR(100),        -- Sightseeing, Culinary, dsb.
    description      TEXT,
    popularity_score INT       DEFAULT 1, -- Meningkat setiap kali tempat ini muncul di plan
    last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk pencarian cepat berdasarkan nama dan lokasi
CREATE INDEX idx_attraction_name_location ON tourist_attractions (name, location_id);

-- Tambahkan ini jika belum ada saat membuat tabel
ALTER TABLE tourist_attractions ADD CONSTRAINT unique_attraction_per_location UNIQUE (name, location_id);