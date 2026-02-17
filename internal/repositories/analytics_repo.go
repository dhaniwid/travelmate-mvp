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

// GetUserStats mengambil statistik lengkap untuk dashboard "Your Impact"
func (r *AnalyticsRepository) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 1. Total trips created from trips table (more reliable than events)
	var totalTrips int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trips 
		WHERE user_id = $1`, userID).Scan(&totalTrips)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get total trips: %w", err)
	}
	stats["total_trips"] = totalTrips

	// 2. Query trips table for aggregate metrics
	var totalDays, uniqueDestinations int
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(trip_days), 0)::int as total_days,
			COUNT(DISTINCT destination)::int as unique_destinations
		FROM trips
		WHERE user_id = $1`, userID).Scan(&totalDays, &uniqueDestinations)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get trip metrics: %w", err)
	}

	stats["total_days"] = totalDays
	stats["unique_destinations"] = uniqueDestinations

	// 3. Calculate derived metrics
	// Hours saved: 2 hours per trip day (planning time saved by AI)
	hoursSaved := totalDays * 2
	stats["hours_saved"] = hoursSaved

	// CO2 saved: 12.5kg per trip (optimized routing reduces carbon footprint)
	co2Saved := float64(totalTrips) * 12.5
	stats["co2_saved"] = co2Saved

	// 4. Conversion metrics (event_type = 'upgrade_clicked')
	var upgradeClicks int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_analytics_events 
		WHERE user_id = $1 AND event_type = 'upgrade_clicked'`, userID).Scan(&upgradeClicks)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get upgrade clicks: %w", err)
	}
	stats["upgrade_clicks"] = upgradeClicks

	// 5. User level based on trips created (gamification)
	userLevel := "Explorer" // Default
	if totalTrips >= 10 {
		userLevel = "Adventurer"
	}
	if totalTrips >= 25 {
		userLevel = "Globetrotter"
	}
	if totalTrips >= 50 {
		userLevel = "Travel Master"
	}
	stats["user_level"] = userLevel

	return stats, nil
}
