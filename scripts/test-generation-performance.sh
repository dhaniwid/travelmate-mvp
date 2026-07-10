#!/bin/bash
# Miru — Trip Generation Performance Test
# Usage: ./scripts/test-generation-performance.sh [CLERK_TOKEN] [DESTINATION] [DAYS]
#
# Examples:
#   ./scripts/test-generation-performance.sh                          # no auth, Yogyakarta 3d
#   ./scripts/test-generation-performance.sh "eyJhb..."              # with Clerk token
#   ./scripts/test-generation-performance.sh "" "Bali" 5             # Bali 5 days, no auth

CLERK_TOKEN=${1:-""}
DESTINATION=${2:-"Yogyakarta"}
TRIP_DAYS=${3:-3}
API_URL="http://localhost:8889"
SERVER_LOG="/tmp/travelmate-server.log"

echo "🧪 Miru Generation Performance Test"
echo "======================================"
echo "  Destination : $DESTINATION"
echo "  Days        : $TRIP_DAYS"
echo "  Auth        : $([ -n "$CLERK_TOKEN" ] && echo 'Clerk token provided' || echo 'no token (guest)')"
echo "  Log file    : $SERVER_LOG"
echo ""

# Mark log position before test so we only grep new lines
LOG_MARK=$(wc -l < "$SERVER_LOG" 2>/dev/null || echo 0)

# Build auth header
AUTH_HEADER=""
if [ -n "$CLERK_TOKEN" ]; then
  AUTH_HEADER="-H \"Authorization: Bearer $CLERK_TOKEN\""
fi

# Record client-side start time
START_MS=$(date +%s%3N)

# Hit generate endpoint
HTTP_CODE=$(curl -s -o /tmp/miru-perf-response.json -w "%{http_code}" \
  -X POST "$API_URL/api/v1/trips" \
  ${CLERK_TOKEN:+-H "Authorization: Bearer $CLERK_TOKEN"} \
  -H "Content-Type: application/json" \
  -d "{
    \"destination\": \"$DESTINATION\",
    \"origin\": \"Anywhere\",
    \"start_date\": \"$(date -v+1d +%Y-%m-%d 2>/dev/null || date -d tomorrow +%Y-%m-%d)\",
    \"trip_days\": $TRIP_DAYS,
    \"style\": \"Balanced mix of rest and activity, Mix of popular spots and local secrets\",
    \"budget\": 0,
    \"user_id\": \"guest\"
  }")

END_MS=$(date +%s%3N)
TOTAL_MS=$((END_MS - START_MS))

echo "📊 Client-side timing:"
echo "  Total response time : ${TOTAL_MS}ms  ($(echo "scale=1; $TOTAL_MS/1000" | bc)s)"
echo "  HTTP status         : $HTTP_CODE"
echo ""

echo "📊 Server-side timing breakdown:"
echo "--------------------------------------"
# Extract only lines added after LOG_MARK
tail -n "+$((LOG_MARK + 1))" "$SERVER_LOG" 2>/dev/null | grep "⏱️ \[GEN" | sed 's/^[0-9\/]* [0-9:]* //'
echo ""

echo "📊 Token usage:"
echo "--------------------------------------"
tail -n "+$((LOG_MARK + 1))" "$SERVER_LOG" 2>/dev/null | grep "completion_tokens" | sed 's/^[0-9\/]* [0-9:]* //'
echo ""

echo "======================================"
if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
  TRIP_ID=$(cat /tmp/miru-perf-response.json | grep -o '"trip_id":"[^"]*"' | head -1 | cut -d'"' -f4)
  echo "✅ Generation SUCCESS"
  echo "  Trip ID: $TRIP_ID"
else
  echo "❌ Generation FAILED"
  echo "  Response: $(cat /tmp/miru-perf-response.json)"
fi
echo "======================================"
