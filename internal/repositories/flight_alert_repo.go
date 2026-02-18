package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"travelmate/internal/domain"
)

// FlightAlertRepository handles database operations for flight price alerts
type FlightAlertRepository struct {
	db *sql.DB
}

// NewFlightAlertRepository creates a new flight alert repository
func NewFlightAlertRepository(db *sql.DB) *FlightAlertRepository {
	return &FlightAlertRepository{db: db}
}

// CreateAlert creates a new flight price alert
func (r *FlightAlertRepository) CreateAlert(ctx context.Context, alert *domain.FlightPriceAlert) error {
	query := `
		INSERT INTO flight_price_alerts (
			trip_id, origin_airport, destination_airport, 
			departure_date, return_date,
			initial_price, current_price, lowest_price_seen,
			currency, last_checked_at, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx, query,
		alert.TripID, alert.OriginAirport, alert.DestinationAirport,
		alert.DepartureDate, alert.ReturnDate,
		alert.InitialPrice, alert.CurrentPrice, alert.LowestPriceSeen,
		alert.Currency, alert.LastCheckedAt, alert.IsActive,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create flight alert: %w", err)
	}

	return nil
}

// GetAlertByTripAndOrigin retrieves alert for a specific trip and origin
func (r *FlightAlertRepository) GetAlertByTripAndOrigin(ctx context.Context, tripID, originAirport string) (*domain.FlightPriceAlert, error) {
	query := `
		SELECT id, trip_id, origin_airport, destination_airport,
		       departure_date, return_date,
		       initial_price, current_price, lowest_price_seen,
		       currency, last_checked_at, is_active,
		       created_at, updated_at
		FROM flight_price_alerts
		WHERE trip_id = $1 AND origin_airport = $2
	`

	var alert domain.FlightPriceAlert
	err := r.db.QueryRowContext(ctx, query, tripID, originAirport).Scan(
		&alert.ID, &alert.TripID, &alert.OriginAirport, &alert.DestinationAirport,
		&alert.DepartureDate, &alert.ReturnDate,
		&alert.InitialPrice, &alert.CurrentPrice, &alert.LowestPriceSeen,
		&alert.Currency, &alert.LastCheckedAt, &alert.IsActive,
		&alert.CreatedAt, &alert.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No alert exists
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get flight alert: %w", err)
	}

	return &alert, nil
}

// GetAlertsByTrip retrieves all alerts for a trip
func (r *FlightAlertRepository) GetAlertsByTrip(ctx context.Context, tripID string) ([]*domain.FlightPriceAlert, error) {
	query := `
		SELECT id, trip_id, origin_airport, destination_airport,
		       departure_date, return_date,
		       initial_price, current_price, lowest_price_seen,
		       currency, last_checked_at, is_active,
		       created_at, updated_at
		FROM flight_price_alerts
		WHERE trip_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query flight alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*domain.FlightPriceAlert
	for rows.Next() {
		var alert domain.FlightPriceAlert
		err := rows.Scan(
			&alert.ID, &alert.TripID, &alert.OriginAirport, &alert.DestinationAirport,
			&alert.DepartureDate, &alert.ReturnDate,
			&alert.InitialPrice, &alert.CurrentPrice, &alert.LowestPriceSeen,
			&alert.Currency, &alert.LastCheckedAt, &alert.IsActive,
			&alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flight alert: %w", err)
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// UpdatePrice updates the current and lowest price for an alert
func (r *FlightAlertRepository) UpdatePrice(ctx context.Context, alertID string, newPrice float64) error {
	query := `
		UPDATE flight_price_alerts
		SET current_price = $1,
		    lowest_price_seen = LEAST(lowest_price_seen, $1),
		    last_checked_at = $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, newPrice, time.Now(), alertID)
	if err != nil {
		return fmt.Errorf("failed to update price: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	return nil
}

// DeactivateAlert sets an alert as inactive
func (r *FlightAlertRepository) DeactivateAlert(ctx context.Context, alertID string) error {
	query := `
		UPDATE flight_price_alerts
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, alertID)
	if err != nil {
		return fmt.Errorf("failed to deactivate alert: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	return nil
}

// GetActiveAlerts retrieves all active alerts (for background worker)
func (r *FlightAlertRepository) GetActiveAlerts(ctx context.Context) ([]*domain.FlightPriceAlert, error) {
	query := `
		SELECT id, trip_id, origin_airport, destination_airport,
		       departure_date, return_date,
		       initial_price, current_price, lowest_price_seen,
		       currency, last_checked_at, is_active,
		       created_at, updated_at
		FROM flight_price_alerts
		WHERE is_active = true
		  AND departure_date >= CURRENT_DATE
		ORDER BY last_checked_at ASC NULLS FIRST
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*domain.FlightPriceAlert
	for rows.Next() {
		var alert domain.FlightPriceAlert
		err := rows.Scan(
			&alert.ID, &alert.TripID, &alert.OriginAirport, &alert.DestinationAirport,
			&alert.DepartureDate, &alert.ReturnDate,
			&alert.InitialPrice, &alert.CurrentPrice, &alert.LowestPriceSeen,
			&alert.Currency, &alert.LastCheckedAt, &alert.IsActive,
			&alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flight alert: %w", err)
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}
