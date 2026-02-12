#!/bin/bash
# Sprint 3 Enrichment Verification Script
# Checks if the enrichment feature is populating the new Phase 2 fields

echo "🔍 Sprint 3 Enrichment Verification"
echo "===================================="
echo ""

# Check for recent trips
echo "📊 Checking most recent trip..."
QUERY="
SELECT 
    id,
    destination,
    created_at,
    plan_data->'itinerary'->0->'activities'->0 as first_activity
FROM trips
WHERE created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC
LIMIT 1;
"

RESULT=$(psql -h localhost -p 5433 -U user -d travelmate -t -c "$QUERY")

if [ -z "$RESULT" ]; then
    echo "❌ No trips found in the last 24 hours"
    echo ""
    echo "💡 To test enrichment:"
    echo "   1. Visit http://localhost:3000"
    echo "   2. Generate a new trip (e.g., Paris, 3 days)"
    echo "   3. Wait 8 seconds for initial generation"
    echo "   4. Wait another 15 seconds for enrichment"
    echo "   5. Run this script again"
    exit 1
fi

echo "✅ Found recent trip"
echo ""
echo "📋 First Activity Details:"
echo "$RESULT" | jq -r '.'
echo ""

# Check for Phase 2 fields
echo "🔎 Checking Phase 2 Enrichment Fields:"
echo "--------------------------------------"

HAS_TYPE=$(echo "$RESULT" | jq -r 'has("type")')
HAS_TIME=$(echo "$RESULT" | jq -r 'has("time")')
HAS_PLACE_ID=$(echo "$RESULT" | jq -r 'has("place_id")')
HAS_IMAGE=$(echo "$RESULT" | jq -r 'has("image_url")')

TYPE_VALUE=$(echo "$RESULT" | jq -r '.type // "MISSING"')
TIME_VALUE=$(echo "$RESULT" | jq -r '.time // "MISSING"')

echo "  📌 Type Field:     $([[ "$HAS_TYPE" == "true" ]] && echo "✅ Present" || echo "❌ Missing") - Value: $TYPE_VALUE"
echo "  ⏰ Time Field:     $([[ "$HAS_TIME" == "true" ]] && echo "✅ Present" || echo "❌ Missing") - Value: $TIME_VALUE"
echo "  🆔 Place ID:       $([[ "$HAS_PLACE_ID" == "true" ]] && echo "✅ Present" || echo "❌ Missing")"
echo "  🖼️  Image URL:      $([[ "$HAS_IMAGE" == "true" ]] && echo "✅ Present" || echo "❌ Missing")"
echo ""

# Check all activities in first day
echo "📅 Checking all activities in Day 1:"
echo "------------------------------------"
ACTIVITIES_QUERY="
SELECT jsonb_array_elements(plan_data->'itinerary'->0->'activities')
FROM trips
WHERE created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC
LIMIT 1;
"

psql -h localhost -p 5433 -U user -d travelmate -t -c "$ACTIVITIES_QUERY" | jq -c '{activity, type, time}' 2>/dev/null || echo "Could not parse activities"

echo ""
echo "✨ Verification Complete!"
