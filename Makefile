# ============================================================
# Miru TravelMate — Go Backend Makefile
# ============================================================

DB_URL ?= $(shell grep '^DATABASE_URL=' .env | cut -d= -f2-)
MIGRATIONS_DIR := migrations
SUPERUSER_URL ?= postgres://postgres@localhost:5432/travelmate?sslmode=disable

.PHONY: migrate migrate-fresh run build test help

## migrate: Run all pending SQL migrations (ordered by filename number)
migrate:
	@echo "🗄️  Running migrations against: $(DB_URL)"
	@if [ -z "$(DB_URL)" ]; then \
		echo "❌  DATABASE_URL not set. Check your .env file."; exit 1; \
	fi
	@ERRORS=0; \
	for f in $$(ls $(MIGRATIONS_DIR)/*.sql | sed 's|.*/||' | sort -t_ -k1,1n | sed "s|^|$(MIGRATIONS_DIR)/|"); do \
		name=$$(basename $$f); \
		result=$$(psql "$(DB_URL)" -f $$f 2>&1); \
		if echo "$$result" | grep -q "^psql:.*ERROR"; then \
			echo "  ❌  $$name"; \
			echo "$$result" | grep "ERROR" | head -1; \
			ERRORS=$$((ERRORS+1)); \
		else \
			echo "  ✅  $$name"; \
		fi; \
	done; \
	if [ $$ERRORS -gt 0 ]; then \
		echo ""; \
		echo "⚠️  $$ERRORS migration(s) had errors."; \
		exit 1; \
	else \
		echo ""; \
		echo "✅  All migrations applied successfully."; \
	fi

## migrate-fresh: DROP all tables and re-run all migrations from scratch (requires superuser)
## Usage: make migrate-fresh SUPERUSER_URL=postgres://postgres@localhost:5432/travelmate?sslmode=disable
migrate-fresh:
	@echo "⚠️  WARNING: This will DROP the entire public schema and recreate it."
	@echo "   Superuser: $(SUPERUSER_URL)"
	@read -p "   Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" != "yes" ]; then echo "Aborted."; exit 1; fi
	@echo "🗑️  Dropping public schema..."
	@psql "$(SUPERUSER_URL)" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" 2>&1
	@psql "$(SUPERUSER_URL)" -c "GRANT ALL ON SCHEMA public TO travelmate_app; GRANT ALL ON SCHEMA public TO public;" 2>&1
	@echo "🗄️  Running all migrations as superuser..."
	@ERRORS=0; \
	for f in $$(ls $(MIGRATIONS_DIR)/*.sql | sed 's|.*/||' | sort -t_ -k1,1n | sed "s|^|$(MIGRATIONS_DIR)/|"); do \
		name=$$(basename $$f); \
		result=$$(psql "$(SUPERUSER_URL)" -f $$f 2>&1); \
		if echo "$$result" | grep -q "^psql:.*ERROR"; then \
			echo "  ❌  $$name"; \
			echo "$$result" | grep "ERROR" | head -1; \
			ERRORS=$$((ERRORS+1)); \
		else \
			echo "  ✅  $$name"; \
		fi; \
	done; \
	psql "$(SUPERUSER_URL)" -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO travelmate_app; GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO travelmate_app; GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO travelmate_app;" > /dev/null 2>&1; \
	if [ $$ERRORS -gt 0 ]; then \
		echo ""; \
		echo "⚠️  $$ERRORS migration(s) had errors (may be harmless if using IF NOT EXISTS)."; \
	else \
		echo ""; \
		echo "✅  Fresh migration complete."; \
	fi

## run: Start the API server
run:
	@echo "🚀  Starting TravelMate API..."
	go run ./cmd/api/main.go

## build: Build the API binary
build:
	@echo "🔨  Building..."
	go build -o bin/api ./cmd/api/main.go
	@echo "✅  Binary at bin/api"

## test: Run all tests
test:
	go test ./... -v

## help: Show this help
help:
	@echo ""
	@echo "Miru TravelMate — Available commands:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  make /'
	@echo ""
