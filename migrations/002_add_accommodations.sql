-- FILE: migrations/002_add_accommodations.sql

-- 1. Create Accommodations Table
CREATE TABLE IF NOT EXISTS accommodations
(
    id              TEXT PRIMARY KEY,               -- UUID
    location_id     TEXT REFERENCES locations (id), -- Relasi ke Kota/Lokasi
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50),                    -- Hotel, Resort, Hostel, Villa
    rating          VARCHAR(10),                    -- "4.5", "5.0"
    price_per_night BIGINT    DEFAULT 0,
    address         TEXT,                           -- Alamat atau area (e.g. "Dago Atas")
    image_url       TEXT,                           -- Persiapan untuk fitur gambar
    description     TEXT,                           -- Deskripsi singkat hotel
    -- Metadata tambahan (Amenities, etc)
    metadata        JSONB     DEFAULT '{}'::JSONB,

    last_updated    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Mencegah duplikasi hotel yang sama di kota yang sama
    UNIQUE (location_id, name)
);

-- 2. UPDATE System Prompt 'planner_system' (THE BRAIN UPGRADE)
-- Kita ubah prompt agar AI memberikan 3 Opsi Transportasi & Data Hotel yang lengkap
UPDATE system_prompts
SET template_text = $$You are an expert travel planner API. Output ONLY valid JSON.

STRICT RULES:
1. TRANSPORT: You MUST provide 3 DISTINCT options if possible:
   - Option 1: Flight (fastest).
   - Option 2: Train (scenic/comfortable).
   - Option 3: Bus/Shuttle/Travel (budget).
   - If a mode is impossible (e.g. Train to Bali), skip it but ensure at least 2 options.
   - For "estimated_time", use format "Xh Ym" (e.g. "2h 15m").

2. ACCOMMODATION: Provide exactly 3 options ranging from Budget to Luxury.
   - "price_per_night": numeric IDR.
   - "location_area": e.g. "City Center", "Dago", "Kuta".

3. BUDGET: Calculate total realistically in IDR.

JSON Schema:
{
  "itinerary": [
    {
      "day": 1,
      "title": "Arrival & Exploration",
      "activities": [
        {"time": "09:00", "activity": "...", "type": "Sightseeing", "description": "..."}
      ]
    }
  ],
  "budget_breakdown": {
    "transport": 0, "accommodation": 0, "food": 0, "tickets": 0, "misc": 0
  },
  "transport_options": [
    {
      "type": "Flight",
      "name": "Garuda Indonesia",
      "price": 1500000,
      "estimated_time": "1h 00m",
      "pros": "Fastest option"
    },
    {
      "type": "Train",
      "name": "Whoosh / Argo Parahyangan",
      "price": 250000,
      "estimated_time": "3h 00m",
      "pros": "Scenic view & comfortable"
    },
    {
      "type": "Bus",
      "name": "DayTrans / Cititrans",
      "price": 110000,
      "estimated_time": "3h 30m",
      "pros": "Budget friendly, many schedules"
    }
  ],
  "accommodation_options": [
    {
      "name": "Hotel Padma",
      "type": "Resort",
      "rating": "4.8",
      "price_per_night": 2500000,
      "location_area": "Ciumbuleuit",
      "description": "Luxury view of the valley"
    }
  ],
  "decision_notes": ["Selected train for balance of cost and comfort"]
}$$,
    version = COALESCE(version, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE key = 'planner_system';