package handlers

import (
	"database/sql"
	"net/http"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	DB *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

type EventBreakdown struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

type AdminStatsResponse struct {
	TotalUsers      int                     `json:"total_users"`
	TotalTrips      int                     `json:"total_trips"`
	PremiumUsers    int                     `json:"premium_users"`
	ConversionRate  float64                 `json:"conversion_rate_pct"`
	EventsLast24h   int                     `json:"events_last_24h"`
	TripsCreated24h int                     `json:"trips_created_24h"`
	NewUsersToday   int                     `json:"new_users_today"`
	RecentActivity  []domain.AnalyticsEvent `json:"recent_activity"`
	EventBreakdown  []EventBreakdown        `json:"event_breakdown"`
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats := AdminStatsResponse{
		RecentActivity: []domain.AnalyticsEvent{},
		EventBreakdown: []EventBreakdown{},
	}

	// 1. Total Users
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// 2. Total Trips
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM trips").Scan(&stats.TotalTrips); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count trips"})
		return
	}

	// 3. Premium Users
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE subscription_tier != 'FREE'").Scan(&stats.PremiumUsers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count premium users"})
		return
	}

	// 4. Conversion Rate
	if stats.TotalUsers > 0 {
		stats.ConversionRate = float64(stats.PremiumUsers) / float64(stats.TotalUsers) * 100
	}

	// 5. Events Last 24h
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_analytics_events WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.EventsLast24h); err == nil {
		// field already set
		_ = stats.EventsLast24h
	}

	// 6. Trips Created 24h
	_ = h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM trips WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.TripsCreated24h)

	// 7. New Users Today (UTC)
	_ = h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.NewUsersToday)

	// 8. Event Breakdown (Top 10 event types, all time)
	breakdownRows, err := h.DB.QueryContext(ctx, `
		SELECT event_type, COUNT(*) AS count
		FROM user_analytics_events
		GROUP BY event_type
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer breakdownRows.Close()
		for breakdownRows.Next() {
			var eb EventBreakdown
			if err := breakdownRows.Scan(&eb.EventType, &eb.Count); err == nil {
				stats.EventBreakdown = append(stats.EventBreakdown, eb)
			}
		}
	}

	// 9. Recent Activity (Last 20 events)
	rows, err := h.DB.QueryContext(ctx, "SELECT id, user_id, event_type, created_at FROM user_analytics_events ORDER BY created_at DESC LIMIT 20")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var evt domain.AnalyticsEvent
			if err := rows.Scan(&evt.ID, &evt.UserID, &evt.EventType, &evt.CreatedAt); err == nil {
				stats.RecentActivity = append(stats.RecentActivity, evt)
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}
