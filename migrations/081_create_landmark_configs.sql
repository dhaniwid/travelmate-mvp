-- UP
CREATE TABLE landmark_configs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug           VARCHAR(100) UNIQUE NOT NULL,
    city_name      VARCHAR(100) NOT NULL,
    landmark_name  VARCHAR(200) NOT NULL,
    landmark_desc  TEXT NOT NULL,
    geo_context    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_landmark_configs_slug ON landmark_configs(slug);

INSERT INTO landmark_configs (slug, city_name, landmark_name, landmark_desc, geo_context) VALUES
    ('bandung', 'Bandung', 'Gedung Sate',
     'a white Dutch colonial government building with distinctive tiered dark green roof tower topped with 6 stacked golden spheres on a vertical rod (sate skewer ornament), arched windows with teal-green frames, symmetrical wings left and right',
     'Silhouette of flat-topped Tangkuban Perahu volcano visible in background center'),
    ('bali', 'Bali', 'Pura Tanah Lot',
     'a traditional Balinese Hindu temple perched on a large offshore rock surrounded by sea, multi-tiered thatched black roofs (meru), surrounded by crashing waves and tropical vegetation',
     'Ocean horizon visible in background, dramatic coastal cliffs on left side'),
    ('jakarta', 'Jakarta', 'Monumen Nasional (Monas)',
     'a tall obelisk monument with a flame-shaped top covered in gold leaf, standing on a large marble base, surrounded by manicured Merdeka Square park with fountain',
     'Modern city skyline silhouette visible in background'),
    ('yogyakarta', 'Yogyakarta', 'Candi Borobudur',
     'a massive 9th-century Buddhist temple, pyramid-shaped with 9 stacked platforms, 72 perforated stupas each containing a Buddha statue, topped with a large central dome stupa, intricate stone carvings throughout',
     'Silhouette of Mount Merapi volcanic cone visible in background, lush tropical jungle surrounding the temple'),
    ('semarang', 'Semarang', 'Lawang Sewu',
     'a cream-white Dutch colonial building with two tall symmetrical gothic towers featuring pointed arched windows and conical pointed spires (NOT domes), large central archway entrance, white and cream facade, dark grey slate roof, colonnaded corridors extending left and right from the towers',
     'Hint of Java Sea coastline visible in far background');

-- DOWN
-- DROP TABLE IF EXISTS landmark_configs;
