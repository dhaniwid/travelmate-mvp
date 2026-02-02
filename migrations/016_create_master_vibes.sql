-- 1. Master Vibes (Nature, City, Culinary, Art)
CREATE TABLE travel_vibes
(
    id        SERIAL PRIMARY KEY,
    name      VARCHAR(50)        NOT NULL, -- e.g. "Nature Escape", "Deep Culture"
    slug      VARCHAR(50) UNIQUE NOT NULL, -- e.g. "nature-escape"
    asset_url TEXT                         -- URL icon/image untuk UI Unity
);

-- 2. Master Destinations (Kota-kota yang kita tawarkan)
CREATE TABLE destinations
(
    id               SERIAL PRIMARY KEY,
    city_name        VARCHAR(100) NOT NULL, -- e.g. "Kyoto", "Bandung"
    country_name     VARCHAR(100) NOT NULL,
    hero_image_url   TEXT,                  -- Gambar utama untuk kartu di Unity
    description      TEXT,                  -- Teaser text
    popularity_score INT DEFAULT 0          -- Untuk sorting, mana yg paling sering diklik
);

-- 3. Junction Table (Hubungan Kota dengan Vibes)
-- Contoh: Bandung punya vibes "Culinary" dan "Nature"
CREATE TABLE destination_vibes
(
    destination_id INT REFERENCES destinations (id),
    vibe_id        INT REFERENCES travel_vibes (id),
    PRIMARY KEY (destination_id, vibe_id)
);

-- 4. Cached Logistics (Hasil Mining Rute)
-- Supaya kita tidak perlu nanya API/LLM terus menerus untuk rute yang sama
CREATE TABLE cached_logistics
(
    id               BIGSERIAL PRIMARY KEY,
    origin_city      VARCHAR(100) NOT NULL,
    destination_city VARCHAR(100) NOT NULL,

    -- Menyimpan JSON mentah hasil generate prompt "Gold Standard" kita
    -- Isinya: Transport options, first mile, last mile, estimasi harga
    route_data       JSONB        NOT NULL,

    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Data logistik bisa basi (harga berubah), set expiry misal 7 hari
    expires_at       TIMESTAMP    NOT NULL
);

-- Indexing untuk pencarian super cepat
CREATE INDEX idx_route_lookup ON cached_logistics (origin_city, destination_city);

-- 5. User Interest Logs
CREATE TABLE user_interest_logs
(
    id                 BIGSERIAL PRIMARY KEY,
    user_id            UUID REFERENCES users (id),       -- Asumsi tabel users sudah ada
    vibe_clicked       INT REFERENCES travel_vibes (id), -- User suka klik "Nature"
    destination_viewed INT REFERENCES destinations (id), -- User kepo sama "Kyoto"
    logged_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE destinations ADD COLUMN IF NOT EXISTS discovery_data JSONB;
ALTER TABLE destinations
    ADD CONSTRAINT unique_city_name UNIQUE (city_name);

-- ==========================================
-- END OF MIGRATION
-- ==========================================