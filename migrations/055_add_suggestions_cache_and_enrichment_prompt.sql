-- migration: 055_add_suggestions_cache_and_enrichment_prompt

-- 1. Add suggestions_cache column to trips table
ALTER TABLE trips ADD COLUMN IF NOT EXISTS suggestions_cache JSONB;

-- 2. Insert the activity_enrichment prompt
INSERT INTO system_prompts (key, template_text, version, description)
VALUES (
    'activity_enrichment',
    'Context: A traveler is in {{.Destination}}. Activity Name: {{.Title}} Based on the activity name, generate a concise and captivating description, a specific place name (if applicable), and a category tag (choose one: sightseeing, culinary, shopping, leisure, or adventure). Return ONLY a JSON object: { "description": "string", "place_name": "string", "latitude": number, "longitude": number, "category": "string", "location_type": "specific|generic" } STRICT LOCATION RULES: 1. GENERIC ACTIVITIES: DO NOT invent specific venue names. COORDINATES: Use center coordinates. FLAG: "generic". 2. SPECIFIC ACTIVITIES: USE real venue names. FLAG: "specific".',
    1,
    'Prompts for enriching manual activity additions with AI sensory details and geocoding.'
)
ON CONFLICT (key) DO UPDATE 
SET template_text = EXCLUDED.template_text,
    version = system_prompts.version + 1,
    updated_at = CURRENT_TIMESTAMP;
