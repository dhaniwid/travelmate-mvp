-- 1. Prompt Utama (Legacy/General)
UPDATE system_prompts
SET template_text = 'Expert Travel API. Output ONLY JSON.
STRICT:
1. NO PLACEHOLDERS: Use real, searchable names (e.g., "Gili Trawangan", not "Beach").
2. ITINERARY: Chronological. "time" format "HH:MM - HH:MM".
3. TRANSPORT: Real operators only (e.g., "Batik Air"). NO trains for island crossings.
4. STAY: 3 real hotels (Budget, Mid, Luxury).
5. JSON SCHEMA: {"itinerary": [], "budget_breakdown": {}, "transport_options": [], "accommodation_options": [], "decision_notes": []}',
    version = COALESCE(version, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE key = 'planner_system';

-- 2. Prompt Itinerary (Spesifik & Cepat)
UPDATE system_prompts
SET
    template_text = 'Generate ONLY the "itinerary" array for this trip.
RULES: Use real venue names. NO placeholders. JSON only.
SCHEMA: {"itinerary": [{"day": 1, "title": "", "activities": [{"time": "", "activity": "", "type": "", "description": "", "place_name": ""}]}]}',
    version = COALESCE(version, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE key = 'planner_itinerary_system';

-- 3. Prompt Logistik (Fokus pada Data)
UPDATE system_prompts
SET template_text = 'Generate ONLY "transport_options", "accommodation_options", and "budget_breakdown".
RULES: Real operators/hotels only. Match trip style/budget. JSON only.
SCHEMA: {"transport_options": [], "accommodation_options": [], "budget_breakdown": {}}',
    version = COALESCE(version, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE key = 'planner_logistics_system';

UPDATE system_prompts
SET template_text = 'Create a {{.TripDays}}-day trip to {{.Destination}} from {{.Origin}}. Style: {{.Style}}. Budget: {{.Budget}}. Start: {{.StartDate}}.',
    version = COALESCE(version, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE key = 'planner_user';