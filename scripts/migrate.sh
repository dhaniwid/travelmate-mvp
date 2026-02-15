#!/bin/bash
set -e

# Miru Database Migration Runner
# ------------------------------
# Usage: ./scripts/migrate.sh "$DATABASE_URL"
#
# This script ensures all .sql files in the migrations/ directory are executed
# in order, tracking progress in a 'schema_migrations' table to avoid duplicates.

DB_URL=$1

if [ -z "$DB_URL" ]; then
    echo "❌ ERROR: No DATABASE_URL provided."
    echo "Usage: bash scripts/migrate.sh \"postgres://user:pass@host:port/db\""
    exit 1
fi

echo "🐘 Connecting to database..."

# 1. Initialize Migration Tracking Table
psql "$DB_URL" -c "CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW());" > /dev/null

# 2. Get List of .sql Files
MIGRATION_DIR="migrations"
FILES=$(ls $MIGRATION_DIR/*.sql | sort)

echo "🚀 Starting migrations from $MIGRATION_DIR/ ..."

APPLIED_COUNT=0
SKIPPED_COUNT=0

for file in $FILES; do
    FILENAME=$(basename "$file")
    
    # Check if version already exists in tracking table
    HAS_VERSION=$(psql "$DB_URL" -tAc "SELECT 1 FROM schema_migrations WHERE version='$FILENAME'")
    
    if [ "$HAS_VERSION" = "1" ]; then
        # echo "⏭️  $FILENAME already applied."
        SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
    else
        echo "🏃 Applying: $FILENAME ..."
        # Execute migration file
        psql "$DB_URL" -f "$file" > /dev/null
        # Record successful migration
        psql "$DB_URL" -c "INSERT INTO schema_migrations (version) VALUES ('$FILENAME');" > /dev/null
        echo "✅ $FILENAME"
        APPLIED_COUNT=$((APPLIED_COUNT + 1))
    fi
done

echo "-------------------------------------------"
echo "🎉 Done! Applied: $APPLIED_COUNT | Skipped: $SKIPPED_COUNT"
echo "-------------------------------------------"
