#!/bin/bash

BASE_URL="${API_URL:-http://localhost:8889/api/v1}"

echo "🚀 Starting Verification: Async Trip Generation (M-123)..."
echo "--------------------------------------------------------"

# 1. POST /trips
echo "1️⃣  Creating Async Trip..."
RESPONSE=$(curl -s -X POST "$BASE_URL/trips" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_2sY...",
    "origin": "Jakarta",
    "destination": "Bandung",
    "trip_days": 3,
    "style": "Relaxed"
  }')

echo "Response Body: $RESPONSE"

# Extract Trip ID (using jq or grep)
TRIP_ID=$(echo $RESPONSE | grep -o 'trip_id":"[^"]*' | cut -d'"' -f3)
STATUS=$(echo $RESPONSE | grep -o 'status":"[^"]*' | cut -d'"' -f3)

if [[ -z "$TRIP_ID" ]]; then
  echo "❌ Failed to create trip. No Trip ID found."
  exit 1
else
  echo "✅ Trip Created! ID: $TRIP_ID"
fi

echo "--------------------------------------------------------"

# 2. GET /trips/:id (Polling)
echo "2️⃣  Checking Trip Status (Immediate)..."
sleep 1
DETAIL_RES=$(curl -s -X GET "$BASE_URL/trips/$TRIP_ID?user_id=user_2sY...")
ENRICH_STATUS=$(echo $DETAIL_RES | grep -o 'enrichment_status":"[^"]*' | cut -d'"' -f3)

echo "Current Status: $ENRICH_STATUS"

if [[ "$ENRICH_STATUS" == "enriching" || "$ENRICH_STATUS" == "completed" ]]; then
    echo "✅ Async Architecture Verified! Status is '$ENRICH_STATUS'"
else
    echo "⚠️  Unexpected Status: '$ENRICH_STATUS' (Expected 'enriching' or 'completed')"
fi

echo "--------------------------------------------------------"
echo "🎉 Verification Complete."
