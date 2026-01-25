UPDATE system_prompts
SET template_text = 'Generate logistics data for this trip.
RULES:
1. Price MUST be a raw integer number (NO quotes, NO currency symbols, NO dots).
2. JSON ONLY.
SCHEMA: {
  "transport_options": [{"type": "", "name": "", "price": 1500000, "estimated_time": "", "pros": ""}],
  "accommodation_options": [{"name": "", "type": "", "rating": "4.5", "price_per_night": 750000, "location_area": "", "description": ""}],
  "budget_breakdown": {"transport": 0, "accommodation": 0, "food": 0, "tickets": 0, "misc": 0}
}'
WHERE key = 'planner_logistics_system';