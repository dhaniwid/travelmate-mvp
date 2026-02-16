-- Migration 054: Add Activity Suggestions Prompt
-- Goal: Provide high-quality activity suggestions for the "Add Activity" feature

UPDATE system_prompts
SET template_text = 'You are a local travel expert. The user is in {{.Destination}} and has a "{{.Style}}" travel style.
    They are looking for activities to add to their itinerary for the {{.Bucket}} bucket (Specific time: {{.Time}}).

    Suggest 5 activities that:
    1. Match the travel style ({{.Style}}).
    2. Are appropriate for the time of day ({{.Bucket}}).
    3. Are authentic local experiences.
    4. Provide a good mix of categories (Culinary, Sightseeing, Shopping, Leisure, Adventure).

    JSON OUTPUT STRUCTURE:
    {
      "alternatives": [
        {
          "activity": "Activity Title",
          "type": "Culinary|Sightseeing|Shopping|Leisure|Adventure",
          "description": "Short description (max 15 words)",
          "place_name": "Specific place name"
        }
      ]
    }',
    description = 'AI-powered activity suggestions for adding new items to itinerary (M-128)',
    updated_at = NOW()
WHERE key = 'add_activity_suggestions';
