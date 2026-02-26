#!/bin/bash

# =============================================================================
# script: run_migrations.sh
# description: Automates the execution of SQL migrations on a blank PostgreSQL DB.
# Usage: DATABASE_URL=postgres://user:pass@host:port/db ./run_migrations.sh
# =============================================================================

# Exit immediately if a command exits with a non-zero status
set -e

# 1. Validation: Ensure DATABASE_URL is provided
if [ -z "$DATABASE_URL" ]; then
    echo "❌ Error: DATABASE_URL environment variable is not set."
    echo "Usage: DATABASE_URL=postgres://user:password@localhost:5432/dbname ./run_migrations.sh"
    exit 1
fi

MIGRATIONS_DIR="migrations"

# 2. Check if migrations directory exists
if [ ! d "$MIGRATIONS_DIR" ]; then
    echo "❌ Error: migrations directory not found in current path."
    exit 1
fi

echo "🚀 Starting migrations on: $(echo $DATABASE_URL | sed 's/:[^:]*@/:****@/')" # Mask password for security
echo "------------------------------------------------------------"

# 3. Collect, Filter and Sort migrations
# - Include *.sql files
# - Exclude .down.sql files
# - Sort alphabetically/numerically
MIGRATION_FILES=$(ls "$MIGRATIONS_DIR"/*.sql | grep -v "\.down\." | sort)

COUNT=0

# 4. Execute migrations
for file in $MIGRATION_FILES; do
    filename=$(basename "$file")
    echo "⌛ Executing: $filename..."
    
    # Run psql with quiet mode and stop on error
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file" > /dev/null
    
    echo "[✓] Executed: $filename"
    COUNT=$((COUNT + 1))
done

echo "------------------------------------------------------------"
echo "✅ Finished! Total migrations executed: $COUNT"
