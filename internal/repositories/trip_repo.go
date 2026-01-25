package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
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
        INSERT INTO trips (id, destination, origin, budget, budget_range, start_date, trip_days, style, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

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

// Parameter ID string
func (r *TripRepository) GetTripWithPlan(ctx context.Context, id string) (*domain.TripAndPlan, error) {
	tripQuery := `SELECT id, destination, origin, budget, start_date, trip_days, style FROM trips WHERE id = $1`

	var trip domain.Trip
	err := r.DB.QueryRowContext(ctx, tripQuery, id).Scan(
		&trip.ID, &trip.Destination, &trip.Origin, &trip.Budget,
		&trip.StartDate, &trip.TripDays, &trip.Style,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	planQuery := `SELECT itinerary, budget_breakdown, transport_options, accommodation_options, decision_notes FROM trip_plans WHERE trip_id = $1`

	var plan domain.TripPlan
	var itinRaw, budgetRaw, transRaw, accomRaw, notesRaw []byte

	err = r.DB.QueryRowContext(ctx, planQuery, id).Scan(
		&itinRaw, &budgetRaw, &transRaw, &accomRaw, &notesRaw,
	)

	if err == nil {
		_ = json.Unmarshal(itinRaw, &plan.Itinerary)
		_ = json.Unmarshal(budgetRaw, &plan.BudgetBreakdown)
		_ = json.Unmarshal(transRaw, &plan.TransportOptions)
		_ = json.Unmarshal(accomRaw, &plan.AccommodationOptions)
		_ = json.Unmarshal(notesRaw, &plan.DecisionNotes)
		plan.TripID = trip.ID
	}

	return &domain.TripAndPlan{Trip: trip, Plan: plan}, nil
}

func (r *TripRepository) GetAllTrips(ctx context.Context) ([]domain.Trip, error) {
	query := `SELECT id, destination, start_date, trip_days, style FROM trips ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []domain.Trip
	for rows.Next() {
		var t domain.Trip
		if err := rows.Scan(&t.ID, &t.Destination, &t.StartDate, &t.TripDays, &t.Style); err != nil {
			continue
		}
		trips = append(trips, t)
	}
	return trips, nil
}

func (r *TripRepository) SaveTripPlan(ctx context.Context, trip domain.Trip, plan domain.TripPlan) error {
	// Memulai transaksi
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback otomatis jika terjadi error sebelum Commit
	defer tx.Rollback()

	// 1. Simpan metadata Trip
	queryTrip := `
        INSERT INTO trips (id, location_id, origin, destination, start_date, trip_days, style, budget, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)`

	_, err = tx.ExecContext(ctx, queryTrip,
		trip.ID, trip.LocationID, trip.Origin, trip.Destination,
		trip.StartDate, trip.TripDays, trip.Style, trip.Budget)
	if err != nil {
		return err
	}

	// 2. Simpan Detail Plan (Itinerary, Transport, Accom dalam bentuk JSON)
	queryPlan := `
        INSERT INTO trip_plans (trip_id, itinerary, budget_breakdown, transport_options, accommodation_options, decision_notes)
        VALUES ($1, $2, $3, $4, $5, $6)`

	itineraryJSON, _ := json.Marshal(plan.Itinerary)
	budgetJSON, _ := json.Marshal(plan.BudgetBreakdown)
	transportJSON, _ := json.Marshal(plan.TransportOptions)
	accomJSON, _ := json.Marshal(plan.AccommodationOptions)
	notesJSON, _ := json.Marshal(plan.DecisionNotes)

	_, err = tx.ExecContext(ctx, queryPlan,
		trip.ID, itineraryJSON, budgetJSON, transportJSON, accomJSON, notesJSON)
	if err != nil {
		return err
	}

	// Commit transaksi jika semua langkah berhasil
	return tx.Commit()
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
		WHERE t.destination ILIKE $1 
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

	return &plan, nil
}
