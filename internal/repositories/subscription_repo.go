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

// GetQuota fetches the trip quota for a specific user and month
// If no quota exists for the month, it initializes it with default generic values (limit 3)
func (r *SubscriptionRepository) GetQuota(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
	query := `
		SELECT trips_created, quota_limit
		FROM trip_quotas
		WHERE user_id = $1 AND month = $2
	`

	var quota domain.TripQuota
	quota.UserID = userID
	quota.Month = month

	err := r.DB.QueryRowContext(ctx, query, userID, month).Scan(&quota.TripsCreated, &quota.QuotaLimit)

	if err == sql.ErrNoRows {
		// Initialize quota for this month
		// Default to 3 for free tier. Service layer can adjust if user is PRO.
		initQuery := `
			INSERT INTO trip_quotas (user_id, month, trips_created, quota_limit)
			VALUES ($1, $2, 0, 3)
			ON CONFLICT (user_id, month) DO NOTHING
			RETURNING trips_created, quota_limit
		`
		err = r.DB.QueryRowContext(ctx, initQuery, userID, month).Scan(&quota.TripsCreated, &quota.QuotaLimit)
		if err != nil {
			// If conflict happened and scan failed, try select again (race condition handle)
			err = r.DB.QueryRowContext(ctx, query, userID, month).Scan(&quota.TripsCreated, &quota.QuotaLimit)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize quota: %w", err)
			}
		}
	} else if err != nil {
		return nil, err
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
