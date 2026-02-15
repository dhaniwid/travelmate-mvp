package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"travelmate/internal/domain"
)

type SubscriptionRepository struct {
	DB *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

func (r *SubscriptionRepository) GetQuota(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
	// First, try to get existing quota record
	query := `
		SELECT trips_created, quota_limit, month
		FROM trip_quotas
		WHERE user_id = $1
	`

	var quota domain.TripQuota
	quota.UserID = userID
	var storedMonth string

	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&quota.TripsCreated, &quota.QuotaLimit, &storedMonth)

	if err == sql.ErrNoRows {
		// Initialize quota for the first time
		initQuery := `
			INSERT INTO trip_quotas (user_id, month, trips_created, quota_limit, last_reset)
			VALUES ($1, $2, 0, 3, NOW())
			ON CONFLICT (user_id) DO UPDATE 
			SET month = EXCLUDED.month, trips_created = 0, last_reset = NOW()
		`
		_, err = r.DB.ExecContext(ctx, initQuery, userID, month)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize quota: %w", err)
		}

		quota.Month = month
		quota.TripsCreated = 0
		quota.QuotaLimit = 3
		return &quota, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch quota: %w", err)
	}

	// Month mismatch check (Reset Logic)
	if storedMonth != month {
		resetQuery := `
			UPDATE trip_quotas
			SET month = $1, trips_created = 0, last_reset = NOW(), updated_at = NOW()
			WHERE user_id = $2
		`
		_, err = r.DB.ExecContext(ctx, resetQuery, month, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to reset monthly quota: %w", err)
		}
		quota.Month = month
		quota.TripsCreated = 0
	} else {
		quota.Month = storedMonth
	}

	return &quota, nil
}

// IncrementQuota increments the trips_created count
func (r *SubscriptionRepository) IncrementQuota(ctx context.Context, userID, month string) error {
	query := `
		UPDATE trip_quotas
		SET trips_created = trips_created + 1, updated_at = NOW()
		WHERE user_id = $1 AND month = $2
	`
	_, err := r.DB.ExecContext(ctx, query, userID, month)
	return err
}

// ResetQuota resets or updates quota limit (e.g. after upgrade)
func (r *SubscriptionRepository) UpdateQuotaLimit(ctx context.Context, userID, month string, newLimit int) error {
	query := `
		UPDATE trip_quotas
		SET quota_limit = $1, updated_at = NOW()
		WHERE user_id = $2 AND month = $3
	`
	_, err := r.DB.ExecContext(ctx, query, newLimit, userID, month)
	return err
}

// LogSubscriptionEvent logs a lifecycle event
func (r *SubscriptionRepository) LogSubscriptionEvent(ctx context.Context, event *domain.SubscriptionEvent) error {
	query := `
		INSERT INTO subscription_events (
			user_id, event_type, from_tier, to_tier, stripe_event_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.DB.ExecContext(ctx, query,
		event.UserID, event.EventType, event.FromTier, event.ToTier,
		event.StripeEventID, event.Metadata,
	)
	return err
}
