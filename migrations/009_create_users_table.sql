-- Tabel User Baru
CREATE TABLE users
(
    id         UUID PRIMARY KEY,           -- Dari Provider Auth (Clerk/Firebase)
    email      VARCHAR(255) NOT NULL,
    plan_type  VARCHAR(20) DEFAULT 'free', -- 'free' or 'premium'
    created_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP
);

-- Update Tabel Trips
ALTER TABLE trips
    ADD COLUMN user_id UUID REFERENCES users (id);
ALTER TABLE trips
    ADD COLUMN is_public BOOLEAN DEFAULT FALSE; -- Untuk fitur "Share my trip"