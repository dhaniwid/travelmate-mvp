-- FILE: migrations/001_init.sql
-- Description: Initial Schema for TravelMate (Complete MVP Version with Enrichment & Routes)

-- ==========================================
-- 1. CORE TABLES (TRIPS & PLANS)
-- ==========================================

CREATE TABLE IF NOT EXISTS trips
(
    id           TEXT PRIMARY KEY,
    location_id  TEXT,
    destination  VARCHAR(255),
    origin       VARCHAR(255),
    budget       BIGINT    DEFAULT 0,
    budget_range VARCHAR(255),
    start_date   VARCHAR(50),
    trip_days    INT,
    style        VARCHAR(50),
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS trip_plans
(
    id                    SERIAL PRIMARY KEY,
    trip_id               TEXT NOT NULL REFERENCES trips (id) ON DELETE CASCADE,

    -- Struktur Data AI (JSONB for flexibility)
    itinerary             JSONB     DEFAULT '[]'::JSONB,
    budget_breakdown      JSONB     DEFAULT '{}'::JSONB,
    transport_options     JSONB     DEFAULT '[]'::JSONB,
    accommodation_options JSONB     DEFAULT '[]'::JSONB,
    decision_notes        JSONB     DEFAULT '[]'::JSONB,

    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- 2. KNOWLEDGE BASE (LOCATIONS & ROUTES)
-- ==========================================

-- Table Locations: Menyimpan data destinasi yang sudah divalidasi/enriched
CREATE TABLE IF NOT EXISTS locations
(
    id           TEXT PRIMARY KEY,
    name         VARCHAR(255) UNIQUE NOT NULL, -- e.g. "Yogyakarta"
    country      VARCHAR(100),
    description  TEXT,
    style_tags   JSONB,                        -- ["Cultural", "Beach", "Adventure"]
    hub_type     VARCHAR(50),                  -- "Airport" / "Station"
    hub_code     VARCHAR(20),                  -- e.g. "YIA" / "GMR"
    hub_name     VARCHAR(255),                 -- e.g. "Yogyakarta International Airport"
    image_url    TEXT,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table Routes: Cache harga & rute antar kota (Mengatasi error 'relation routes does not exist')
CREATE TABLE routes
(
    id                TEXT PRIMARY KEY,
    origin_code       VARCHAR(10) NOT NULL,
    destination_code  VARCHAR(10) NOT NULL,
    transport_mode    VARCHAR(50),
    provider_name     VARCHAR(100),
    price             BIGINT    DEFAULT 0,
    avg_duration_mins INT       DEFAULT 0,
    last_updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Mencegah duplikasi
    UNIQUE (origin_code, destination_code, transport_mode, provider_name)
);

CREATE UNIQUE INDEX IF NOT EXISTS routes_learning_idx
    ON routes (origin_code, destination_code, transport_mode);

-- ==========================================
-- 3. SYSTEM CONFIG (PROMPTS)
-- ==========================================

CREATE TABLE IF NOT EXISTS system_prompts
(
    id            SERIAL PRIMARY KEY,
    key           VARCHAR(100) UNIQUE NOT NULL,
    template_text TEXT                NOT NULL,
    description   TEXT,
    version       INT       DEFAULT 1,
    is_active     BOOLEAN   DEFAULT TRUE,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- 4. USER DATA (FEEDBACK)
-- ==========================================

CREATE TABLE IF NOT EXISTS feedbacks
(
    id         TEXT PRIMARY KEY,
    trip_id    TEXT REFERENCES trips (id),
    rating     INT CHECK (rating >= 1 AND rating <= 5),
    comment    TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Membuat tabel master untuk Bandara/Stasiun
CREATE TABLE transport_hubs
(
    id          TEXT PRIMARY KEY,            -- Menggunakan TEXT agar support UUID dari Go
    location_id TEXT,
    code        VARCHAR(20) UNIQUE NOT NULL, -- Kode IATA/Stasiun (misal: CGK, GMR)
    name        VARCHAR(255),                -- Nama lengkap (misal: Soekarno Hatta Intl Airport)
    type        VARCHAR(50),                 -- Tipe (Airport / Train Station / Bus Terminal)
    city        VARCHAR(100),                -- Kota lokasi hub
    country     VARCHAR(100),                -- Negara
    coordinates VARCHAR(100),                -- Latitude/Longitude (jika ada)
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- 5. SEEDING DATA (INITIAL PROMPTS)
-- ==========================================
-- A. Prompt System (Otak Utama)
INSERT INTO system_prompts (key, description, template_text)
VALUES ('planner_system',
        'Main system prompt for travel planning with strict budget and transport rules',
        $$You are an expert travel planner API. You must output ONLY valid JSON. No markdown.

STRICT RULES FOR LOGIC:
1. BUDGET REALISM:
   - If user budget is 0 (Calculate for me), you MUST estimate REALISTIC prices based on current economic data.
   - International flights (e.g. Jakarta -> Japan/Europe) typically cost IDR 5.000.000 - 15.000.000+ per person.
   - Do NOT output impossibly low prices.
   - Accommodation in major tourist cities usually starts from IDR 500.000/night for budget.

2. TRANSPORT LOGIC:
   - For INTER-ISLAND or INTERNATIONAL trips, you MUST suggest "Flight".
   - DO NOT suggest "Bus" or "Car" for crossing oceans unless a ferry is standard.
   - For Japan trips, ALWAYS suggest "Train" (Shinkansen) as an option.
   - Prices must be numeric integers (IDR).

3. ACCOMMODATION OUTPUT:
   - You MUST provide exactly 3 options in the "accommodation_options" array.
   - Provide specific real hotel names (e.g. "Apa Hotel Shinjuku", "Hilton Tokyo").
   - "price_per_night" must be a number.

JSON Schema:
{
  "itinerary": [
    {
      "day": 1,
      "title": "Theme of the day",
      "activities": ["Activity 1", "Activity 2"]
    }
  ],
  "budget_breakdown": {
    "transport": 0,
    "accommodation": 0,
    "food": 0,
    "tickets": 0,
    "misc": 0
  },
  "transport_options": [
    {
      "type": "string",
      "name": "string",
      "price": 0,
      "estimated_time": "string",
      "pros": "string"
    }
  ],
  "accommodation_options": [
    {
      "name": "string",
      "type": "string",
      "rating": "string",
      "price_per_night": 0,
      "location_note": "string"
    }
  ],
  "decision_notes": ["string"]
}$$)
ON CONFLICT (key) DO NOTHING;

-- B. Prompt User (Template Request)
INSERT INTO system_prompts (key, description, template_text)
VALUES ('planner_user',
        'User prompt template constructed from request data',
        $$Create a {{.Days}}-day trip to {{.Destination}} from {{.Origin}}.
Style: {{.Style}} (If "General", provide a balanced mix of top iconic attractions and local culture).
Budget Constraint: {{.Budget}} (If 0, assume standard tourist pricing).
Start Date: {{.StartDate}}.
Transport Context: {{.TransportContext}}$$)
ON CONFLICT (key) DO NOTHING;

-- C. ENRICHMENT SYSTEM PROMPT (Fix Error 'enrichment_system not found')
INSERT INTO system_prompts (key, description, template_text)
VALUES ('enrichment_system',
        'Prompt to clean location names and find transport hubs',
        $$You are a Location Data Enrichment API.
INPUT: Raw location string (e.g. "Jogja", "Bali", "Japan").
OUTPUT: Valid JSON matching exactly this structure.

JSON Schema:
{
  "name": "Yogyakarta",
  "country": "Indonesia",
  "description": "A cultural hub known for temples and arts.",
  "styles": ["Culture", "History", "Nature"],
  "hub_type": "Airport",
  "hub_code": "YIA",
  "hub_name": "Yogyakarta International Airport"
}

RULES:
1. "hub_type" must be either "Airport" or "Station".
2. "hub_code" must be the 3-letter IATA code (for Airport) or Station Code.
3. If the location is a country, return the capital city's hub.
$$)
ON CONFLICT (key) DO NOTHING;

-- D. ENRICHMENT USER PROMPT (Pasangan dari enrichment_system)
-- Ini diperlukan agar LocationService bisa memasukkan nama kota ke dalam prompt
INSERT INTO system_prompts (key, description, template_text)
VALUES ('enrichment_user',
        'Template for sending raw location to enrichment AI',
        $$Please analyze and enrich the location data for: "{{.Location}}".
Return the JSON object as specified in the system prompt.$$)
ON CONFLICT (key) DO NOTHING;

-- ==========================================
INSERT INTO system_prompts (key, template_text, version)
VALUES ('planner_city_guide_system',
        'You are a Travel Journalist & Local Guide.
            Goal: Sell the destination "{{.Destination}}" to a potential traveler. Inspire them!

            OUTPUT FORMAT: JSON

            STRUCTURE:
            1. tagline: Catchy 5-7 words summary.
            2. vibes: Array of 3-4 keywords (e.g. "Spicy Food", "Chill", "History").
            3. highlights:
               - Top 3 "Must Visit" spots (Name, Type, Why it''s cool).
            4. culinary_signature:
               - Top 2 dishes user MUST try (Name, Description, Best time to eat).
            5. hidden_gem:
               - 1 place/activity that is underrated or less touristy.
            6. history_snippet:
               - 1 sentence fun fact about the city''s past.

            TONE: Enthusiastic, Insightful, Evocative.',
        1);