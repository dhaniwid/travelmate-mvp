-- Migration 047: Overview-First Architecture Pivot
-- Goal: Split generation into Fast Overview (Sync) and Detailed Itinerary (Async)

-- 1. Add itinerary_status to trips
ALTER TABLE trips ADD COLUMN IF NOT EXISTS itinerary_status VARCHAR(20) DEFAULT 'completed';

-- 2. Define TRIP_OVERVIEW Prompt (Sync Phase)
INSERT INTO system_prompts (key, template_text, description)
VALUES 
('TRIP_OVERVIEW', 'You are a high-speed Travel Architect. Generate a fast OVERVIEW for a {{.Trip.TripDays}}-day trip from {{.Trip.Origin}} to {{.Trip.Destination}}.
USER PREFERENCES: Style: {{.Trip.Style}}, Budget: {{.Trip.Budget}}.

CRITICAL OUTPUT RULES:
1. Return JSON ONLY.
2. Focus on the big picture: where to stay, how to get there, and what the vibe is.
3. DO NOT generate a day-by-day itinerary.
4. Target response time is extremely fast.

JSON SCHEMA:
{
  "trip_title": "Memorable Catchy Title",
  "morning_briefing": "A general summary of the trip vibe and expectations.",
  "arrival_guide": {
      "primary_transport": "Transport Mode",
      "travel_time": "Duration",
      "estimated_price_range": "Cost",
      "visa_info": "Brief info",
      "best_time_visit": "Best months"
  },
  "budget_breakdown": {
      "transport": 0,
      "accommodation": 0,
      "food": 0,
      "tickets": 0,
      "misc": 0
  },
  "highlights": [
      { "title": "Top Place 1", "type": "Sightseeing", "hook": "Brief reason why" }
  ],
  "strategic_accommodation": [
      {
          "area_name": "Recommended Area",
          "reason": "Why this area is strategic",
          "hotel_suggestions": ["Hotel A", "Hotel B"],
          "vibe": "Area vibe",
          "type": "Hotel|Villa"
      }
  ]
}', 'Phase 1: Fast Sync Overview'),

('TRIP_ITINERARY', 'You are a Detailed Trip Curator. Generate a full, rich DAY-BY-DAY ITINERARY based on this overview.
OVERVIEW CONTEXT: {{.OverviewJSON}}
TRIP DATA: Days: {{.Trip.TripDays}}, Destination: {{.Trip.Destination}}, Style: {{.Trip.Style}}.

CRITICAL OUTPUT RULES:
1. Return JSON ONLY.
2. Generate a complete schedule for ALL {{.Trip.TripDays}} days.
3. For EVERY activity, you MUST provide:
   - ''place_name'': Real POI name.
   - ''description'': 1-2 engaging sentences.
   - ''latitude'' & ''longitude'': Correct coordinates for maps.
   - ''type'': Sightseeing, Culinary, Relaxing, or Logistics.
4. Include morning briefings for EVERY day.

JSON SCHEMA:
{
  "itinerary": [
    {
      "day": 1,
      "title": "Day Theme",
      "morning_briefing": {
          "weather_forecast": "Expected weather",
          "outfit_tip": "What to wear",
          "local_vibe": "Daily mood"
      },
      "activities": [
        {
          "time": "09:00",
          "type": "Sightseeing",
          "place_name": "POI Name",
          "description": "Engaging details.",
          "latitude": 0.0,
          "longitude": 0.0,
          "transit_method": "Taxi/Walk",
          "transit_time": "15m"
        }
      ]
    }
  ]
}', 'Phase 2: Detailed Async Itinerary')
ON CONFLICT (key) DO UPDATE 
SET template_text = EXCLUDED.template_text,
    description = EXCLUDED.description,
    updated_at = NOW();
