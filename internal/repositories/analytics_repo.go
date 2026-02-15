package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"travelmate/internal/domain"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// SaveEvent menyimpan event analitik baru ke database
func (r *AnalyticsRepository) SaveEvent(ctx context.Context, event domain.AnalyticsEvent) error {
	dataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO user_analytics_events (
			user_id,
			event_type,
			event_data
		) VALUES ($1, $2, $3)
	`

	_, err = r.db.ExecContext(ctx, query, event.UserID, event.EventType, dataJSON)
	if err != nil {
		return fmt.Errorf("failed to insert analytics event: %w", err)
	}

	return nil
}

// GetUserStats mengambil statistik dasar untuk dashboard "Your Impact"
func (r *AnalyticsRepository) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 1. Total trips created (event_type = 'trip_success')
	var totalTrips int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_analytics_events 
		WHERE user_id = $1 AND event_type = 'trip_success'`, userID).Scan(&totalTrips)
	if err != nil {
		return nil, err
	}
	stats["total_trips"] = totalTrips

	// 2. Conversion clicks (event_type = 'upgrade_clicked')
	var upgradeClicks int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_analytics_events 
		WHERE user_id = $1 AND event_type = 'upgrade_clicked'`, userID).Scan(&upgradeClicks)
	if err != nil {
		return nil, err
	}
	stats["upgrade_clicks"] = upgradeClicks

	return stats, nil
}
