package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"travelmate/internal/domain"
)

type TripRepository struct {
	DB *sql.DB
}

func NewTripRepository(db *sql.DB) *TripRepository {
	return &TripRepository{DB: db}
}

// CreateTrip sekarang hanya menerima data yang sudah punya ID (dari Service)
func (r *TripRepository) CreateTrip(ctx context.Context, trip *domain.Trip) error {
	query := `
        INSERT INTO trips (id, destination, origin, budget, budget_range, start_date, trip_days, style, created_at, ai_edits_used)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	// Kita kirim trip.ID yang sudah digenerate di Service
	_, err := r.DB.ExecContext(ctx, query,
		trip.ID,
		trip.Destination,
		trip.Origin,
		trip.Budget,
		trip.BudgetRange,
		trip.StartDate,
		trip.TripDays,
		trip.Style,
		trip.CreatedAt,
		trip.AIEditsUsed,
	)

	return err
}

func (r *TripRepository) SavePlan(ctx context.Context, plan domain.TripPlan) error {
	itineraryJSON, _ := json.Marshal(plan.Itinerary)
	transportJSON, _ := json.Marshal(plan.TransportOptions)
	accomJSON, _ := json.Marshal(plan.AccommodationOptions)
	budgetJSON, _ := json.Marshal(plan.BudgetBreakdown)
	notesJSON, _ := json.Marshal(plan.DecisionNotes)

	query := `
        INSERT INTO trip_plans (trip_id, itinerary, budget_breakdown, transport_options, accommodation_options, decision_notes)
        VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.DB.ExecContext(ctx, query,
		plan.TripID, // String
		itineraryJSON,
		budgetJSON,
		transportJSON,
		accomJSON,
		notesJSON,
	)
	return err
}

func (r *TripRepository) GetTripWithPlan(ctx context.Context, id string) (*domain.TripAndPlan, error) {
	query := `
        SELECT 
            id, user_id, location_id, destination, origin, 
            start_date, trip_days, style, budget, budget_range, 
            is_public, created_at, plan_data, status, enrichment_status, itinerary_status, ai_edits_used, suggestions_cache
        FROM trips
        WHERE id = $1
    `

	var trip domain.Trip
	var planDataRaw []byte

	var userID sql.NullString
	var locationID sql.NullString
	var budgetRange sql.NullString
	var enrichmentStatus sql.NullString
	var itineraryStatus sql.NullString

	var suggestionsCacheRaw []byte

	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&trip.ID,
		&userID,
		&locationID,
		&trip.Destination,
		&trip.Origin,
		&trip.StartDate,
		&trip.TripDays,
		&trip.Style,
		&trip.Budget,
		&budgetRange,
		&trip.IsPublic,
		&trip.CreatedAt,
		&planDataRaw,
		&trip.Status,
		&enrichmentStatus,
		&itineraryStatus,
		&trip.AIEditsUsed,
		&suggestionsCacheRaw,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan trip: %w", err)
	}

	trip.SuggestionsCache = json.RawMessage(suggestionsCacheRaw)

	// Mapping NullString ke Struct
	if userID.Valid {
		trip.UserID = userID.String
	}
	if locationID.Valid {
		trip.LocationID = locationID.String
	}

	if budgetRange.Valid {
		trip.BudgetRange = budgetRange.String
	}
	if enrichmentStatus.Valid {
		trip.EnrichmentStatus = enrichmentStatus.String
	}
	if itineraryStatus.Valid {
		trip.ItineraryStatus = itineraryStatus.String
	}

	var plan domain.TripPlan
	if len(planDataRaw) > 0 {
		// PEEK: Check if it's double-encoded (starts with double quote)
		// This happens if data was stringified twice before save.
		if len(planDataRaw) > 2 && planDataRaw[0] == '"' {
			var unquoted string
			if err := json.Unmarshal(planDataRaw, &unquoted); err == nil {
				planDataRaw = []byte(unquoted)
			}
		}

		if err := json.Unmarshal(planDataRaw, &plan); err != nil {
			log.Printf("⚠️ Warning: Failed to unmarshal plan_data for trip %s: %v. Attempting soft recovery...", id, err)

			// RECOVERY MODE: Try to treat it as a generic map and salvage parts
			var rawMap map[string]json.RawMessage
			if errRaw := json.Unmarshal(planDataRaw, &rawMap); errRaw == nil {
				// 1. Try to recover Itinerary
				if itinRaw, ok := rawMap["itinerary"]; ok {
					if errItin := json.Unmarshal(itinRaw, &plan.Itinerary); errItin != nil {
						log.Printf("Full itinerary recovery failed: %v. Trying partial recovery...", errItin)
						// Optional: Try to recover day by day if Itinerary is array
						var itinDays []json.RawMessage
						if errArr := json.Unmarshal(itinRaw, &itinDays); errArr == nil {
							for _, dayRaw := range itinDays {
								var day domain.ItineraryDay
								if errDay := json.Unmarshal(dayRaw, &day); errDay == nil {
									plan.Itinerary = append(plan.Itinerary, day)
								}
							}
						}
					}
				}

				// 2. Recover other components independently
				if budgetRaw, ok := rawMap["budget_breakdown"]; ok {
					_ = json.Unmarshal(budgetRaw, &plan.BudgetBreakdown) // Best effort
				}

				if transRaw, ok := rawMap["transport_options"]; ok {
					_ = json.Unmarshal(transRaw, &plan.TransportOptions)
				}

				// Note: Field matches json tag in models.go
				if accomRaw, ok := rawMap["strategic_accommodation"]; ok {
					_ = json.Unmarshal(accomRaw, &plan.AccommodationOptions)
				}

				if notesRaw, ok := rawMap["decision_notes"]; ok {
					_ = json.Unmarshal(notesRaw, &plan.DecisionNotes)
				}

				// Ensure at least basic structural integrity
				log.Printf("♻️ Soft recovery completed for Trip %s. Itinerary days recovered: %d", id, len(plan.Itinerary))
			} else {
				log.Printf("❌ Critical: Raw JSON structure completely invalid for Trip %s: %v", id, errRaw)
			}
		}
	} else {
		// FALLBACK: Jika plan_data di tabel trips kosong, coba cari di tabel legacy trip_plans
		queryLegacy := `
            SELECT 
                trip_id, itinerary, budget_breakdown, transport_options, accommodation_options, decision_notes
            FROM trip_plans
            WHERE trip_id = $1
        `
		var itiJSON, budgetJSON, transportJSON, accomJSON, notesJSON []byte
		errLegacy := r.DB.QueryRowContext(ctx, queryLegacy, id).Scan(
			&plan.TripID, &itiJSON, &budgetJSON, &transportJSON, &accomJSON, &notesJSON,
		)
		if errLegacy == nil {
			_ = json.Unmarshal(itiJSON, &plan.Itinerary)
			_ = json.Unmarshal(budgetJSON, &plan.BudgetBreakdown)
			_ = json.Unmarshal(transportJSON, &plan.TransportOptions)
			_ = json.Unmarshal(accomJSON, &plan.AccommodationOptions)
			_ = json.Unmarshal(notesJSON, &plan.DecisionNotes)
		}
	}

	// Double check TripID population
	if plan.TripID == "" {
		plan.TripID = id
	}

	return &domain.TripAndPlan{
		Trip: trip,
		Plan: plan,
	}, nil
}

func (r *TripRepository) GetAllTrips(ctx context.Context) ([]domain.Trip, error) {
	query := `SELECT id, destination, start_date, trip_days, style, status, enrichment_status, itinerary_status FROM trips ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []domain.Trip
	for rows.Next() {
		var t domain.Trip
		var enrichmentStatus sql.NullString
		var itineraryStatus sql.NullString
		if err := rows.Scan(&t.ID, &t.Destination, &t.StartDate, &t.TripDays, &t.Style, &t.Status, &enrichmentStatus, &itineraryStatus); err != nil {
			continue
		}
		if enrichmentStatus.Valid {
			t.EnrichmentStatus = enrichmentStatus.String
		}
		if itineraryStatus.Valid {
			t.ItineraryStatus = itineraryStatus.String
		}
		trips = append(trips, t)
	}
	return trips, nil
}

func (r *TripRepository) SaveTripPlan(ctx context.Context, trip domain.Trip, plan domain.TripPlan) error {
	log.Printf("DEBUG: Arrival Guide Present? %v | Packing List: %d items", plan.ArrivalGuide != nil, len(plan.PackingList))

	// 1. Convert struct Plan menjadi JSON string
	planJson, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan data: %w", err)
	}

	// 2. Simpan atau Update metadata Trip beserta plan_data-nya langsung di tabel 'trips'
	// Kita gunakan UPSERT (ON CONFLICT) agar jika trip sudah ada (misal dari CreateTrip)
	// maka plan_data-nya terupdate.
	query := `
        INSERT INTO trips (
            id, user_id, location_id, origin, destination, start_date, trip_days, style, budget, plan_data, enrichment_status, itinerary_status, ai_edits_used, suggestions_cache, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, CURRENT_TIMESTAMP
        )
        ON CONFLICT (id) DO UPDATE SET 
            plan_data = EXCLUDED.plan_data,
            location_id = EXCLUDED.location_id,
            enrichment_status = EXCLUDED.enrichment_status,
            itinerary_status = EXCLUDED.itinerary_status,
            ai_edits_used = EXCLUDED.ai_edits_used,
            suggestions_cache = EXCLUDED.suggestions_cache,
            user_id = COALESCE(NULLIF(EXCLUDED.user_id, ''), trips.user_id)
    `

	// Defensive check for SuggestionsCache (prevent pq: invalid input syntax for type json)
	var suggestionsCache interface{} = trip.SuggestionsCache
	if len(trip.SuggestionsCache) == 0 {
		suggestionsCache = nil
	}

	_, err = r.DB.ExecContext(ctx, query,
		trip.ID, trip.UserID, trip.LocationID, trip.Origin, trip.Destination,
		trip.StartDate, trip.TripDays, trip.Style, trip.Budget, planJson,
		trip.EnrichmentStatus, trip.ItineraryStatus, trip.AIEditsUsed, suggestionsCache)

	return err
}

func (r *TripRepository) GetExistingPlanByCriteria(ctx context.Context, destination, style string, days int) (*domain.TripPlan, error) {
	query := `
		SELECT 
			tp.trip_id, 
			tp.itinerary, 
			tp.budget_breakdown, 
			tp.transport_options, 
			tp.accommodation_options, 
			tp.decision_notes
		FROM trip_plans tp
		JOIN trips t ON tp.trip_id = t.id
		WHERE LOWER(t.destination) = LOWER($1)
		  AND t.style = $2 
		  AND t.trip_days = $3
		ORDER BY t.created_at DESC
		LIMIT 1
	`

	var plan domain.TripPlan
	var itineraryJSON, budgetJSON, transportJSON, accomJSON, decisionNotes []byte

	err := r.DB.QueryRowContext(ctx, query, destination, style, days).Scan(
		&plan.TripID,
		&itineraryJSON,
		&budgetJSON,
		&transportJSON,
		&accomJSON,
		&decisionNotes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(itineraryJSON, &plan.Itinerary)
	_ = json.Unmarshal(budgetJSON, &plan.BudgetBreakdown)
	_ = json.Unmarshal(transportJSON, &plan.TransportOptions)
	_ = json.Unmarshal(accomJSON, &plan.AccommodationOptions)
	_ = json.Unmarshal(decisionNotes, &plan.DecisionNotes)

	if plan.TripID == "" {
		// Try to fallback if not scanned
	}

	return &plan, nil
}

func (r *TripRepository) Create(ctx context.Context, trip *domain.Trip) error {
	// 1. Convert struct PlanData menjadi JSON string untuk disimpan di kolom JSONB
	planJson, err := json.Marshal(trip.PlanData)
	if err != nil {
		return fmt.Errorf("failed to marshal plan data: %w", err)
	}

	query := `
		INSERT INTO trips (
			id, user_id, location_id, destination, origin, 
			start_date, trip_days, style, budget, budget_range, 
			plan_data, enrichment_status, itinerary_status, ai_edits_used, suggestions_cache, created_at
		) VALUES (
			$1, $2, $3, $4, $5, 
			$6, $7, $8, $9, $10, 
			$11, $12, $13, $14, $15, NOW()
		)
	`
	log.Printf("DEBUG REPO CREATE: Trip %s EnrichmentStatus: '%s'", trip.ID, trip.EnrichmentStatus)

	// Defensive check for SuggestionsCache
	var suggestionsCache interface{} = trip.SuggestionsCache
	if len(trip.SuggestionsCache) == 0 {
		suggestionsCache = nil
	}

	_, err = r.DB.ExecContext(ctx, query,
		trip.ID,               // $1
		trip.UserID,           // $2
		trip.LocationID,       // $3
		trip.Destination,      // $4
		trip.Origin,           // $5
		trip.StartDate,        // $6
		trip.TripDays,         // $7
		trip.Style,            // $8
		trip.Budget,           // $9
		trip.BudgetRange,      // $10
		planJson,              // $11
		trip.EnrichmentStatus, // $12
		trip.ItineraryStatus,  // $13
		trip.AIEditsUsed,      // $14
		suggestionsCache,      // $15
	)

	if err != nil {
		return fmt.Errorf("failed to insert trip: %w", err)
	}

	return nil
}

func (r *TripRepository) Delete(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM trips WHERE id = $1 AND user_id = $2`

	result, err := r.DB.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// GetByID GetByID: Mengambil metadata trip saja (Ringan, tanpa plan_data)
// Now supports collaboration - fetches collaborators if requested
func (r *TripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	query := `
        SELECT 
            id, user_id, location_id, destination, origin, 
            start_date, trip_days, style, budget, budget_range, 
            is_public, created_at, status, enrichment_status, itinerary_status, ai_edits_used, suggestions_cache
        FROM trips
        WHERE id = $1
    `

	var trip domain.Trip
	var enrichmentStatus sql.NullString
	var itineraryStatus sql.NullString
	var suggestionsCacheRaw []byte

	// Perhatikan: Tidak ada scan ke &planDataRaw
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.LocationID,
		&trip.Destination,
		&trip.Origin,
		&trip.StartDate,
		&trip.TripDays,
		&trip.Style,
		&trip.Budget,
		&trip.BudgetRange,
		&trip.IsPublic,
		&trip.CreatedAt,
		&trip.Status,
		&enrichmentStatus,
		&itineraryStatus,
		&trip.AIEditsUsed,
		&suggestionsCacheRaw,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trip not found")
	}
	if err != nil {
		return nil, err
	}

	trip.SuggestionsCache = json.RawMessage(suggestionsCacheRaw)

	if enrichmentStatus.Valid {
		trip.EnrichmentStatus = enrichmentStatus.String
	}
	if itineraryStatus.Valid {
		trip.ItineraryStatus = itineraryStatus.String
	}

	// Fetch collaborators for this trip
	collaborators, err := r.getCollaboratorsForTrip(ctx, id)
	if err != nil {
		log.Printf("Warning: failed to fetch collaborators for trip %s: %v", id, err)
		// Don't fail the whole request, just log the error
	} else {
		trip.Collaborators = collaborators
	}

	return &trip, nil
}

// ListTripsByUser mengambil semua trip milik user tertentu
// Now includes trips where user is a collaborator
func (r *TripRepository) ListTripsByUser(ctx context.Context, userID string) ([]domain.Trip, error) {
	// 🔍 FILTER BY USER_ID OR COLLABORATOR ACCESS
	// This query returns trips where the user is either:
	// 1. The owner (trips.user_id = userID)
	// 2. An accepted collaborator
	query := `
        SELECT DISTINCT
            t.id, t.user_id, t.location_id, t.destination, t.origin, 
            t.start_date, t.trip_days, t.style,  
            t.is_public, t.created_at, t.status, t.enrichment_status, t.itinerary_status
        FROM trips t
        LEFT JOIN trip_collaborators tc ON t.id = tc.trip_id
        WHERE t.user_id = $1 
           OR (tc.user_id = $1 AND tc.status = 'accepted')
        ORDER BY t.created_at DESC
    `

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []domain.Trip
	for rows.Next() {
		var t domain.Trip
		// Helper vars untuk handle nullable jika perlu
		var locID sql.NullString
		var enrichmentStatus sql.NullString
		var itineraryStatus sql.NullString

		err := rows.Scan(
			&t.ID, &t.UserID, &locID, &t.Destination, &t.Origin,
			&t.StartDate, &t.TripDays, &t.Style,
			&t.IsPublic, &t.CreatedAt, &t.Status, &enrichmentStatus, &itineraryStatus,
		)
		if err != nil {
			return nil, err
		}

		if locID.Valid {
			t.LocationID = locID.String
		}
		if enrichmentStatus.Valid {
			t.EnrichmentStatus = enrichmentStatus.String
		}
		if itineraryStatus.Valid {
			t.ItineraryStatus = itineraryStatus.String
		}

		trips = append(trips, t)
	}

	return trips, nil
}

// [NEW] Fungsi untuk meng-klaim trip yang statusnya masih DRAFT/Generated
func (r *TripRepository) ClaimTrip(ctx context.Context, tripID string, userID string, planData *domain.TripPlan) error {
	var planJson []byte
	var err error

	if planData != nil {
		planJson, err = json.Marshal(planData)
		if err != nil {
			return fmt.Errorf("failed to marshal plan data: %w", err)
		}
	}

	query := `
        UPDATE trips 
        SET 
            user_id = $1, 
            status = 'UPCOMING',
            plan_data = COALESCE(NULLIF($2, 'null'::jsonb), plan_data),
            updated_at = NOW()
        WHERE id = $3
    `

	result, err := r.DB.ExecContext(ctx, query, userID, planJson, tripID)
	if err != nil {
		return fmt.Errorf("failed to claim trip: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("trip not found with id: %s", tripID)
	}

	return nil
}

// CountUserTrips counts trips created by a user
func (r *TripRepository) CountUserTrips(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM trips WHERE user_id = $1`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// getCollaboratorsForTrip is a helper method to fetch collaborators for a trip
func (r *TripRepository) getCollaboratorsForTrip(ctx context.Context, tripID string) ([]domain.Collaborator, error) {
	query := `
		SELECT id, trip_id, user_id, role, status, invited_by, created_at, updated_at
		FROM trip_collaborators
		WHERE trip_id = $1
		ORDER BY 
			CASE role 
				WHEN 'owner' THEN 1
				WHEN 'editor' THEN 2
				WHEN 'viewer' THEN 3
			END,
			created_at ASC
	`

	rows, err := r.DB.QueryContext(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query collaborators: %w", err)
	}
	defer rows.Close()

	var collaborators []domain.Collaborator
	for rows.Next() {
		var c domain.Collaborator
		err := rows.Scan(
			&c.ID,
			&c.TripID,
			&c.UserID,
			&c.Role,
			&c.Status,
			&c.InvitedBy,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collaborator: %w", err)
		}
		collaborators = append(collaborators, c)
	}

	return collaborators, nil
}
