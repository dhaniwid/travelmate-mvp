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

type AdminStatsResponse struct {
	TotalUsers      int                     `json:"total_users"`
	TotalTrips      int                     `json:"total_trips"`
	PremiumUsers    int                     `json:"premium_users"`
	EventsLast24h   int                     `json:"events_last_24h"`
	TripsCreated24h int                     `json:"trips_created_24h"`
	RecentActivity  []domain.AnalyticsEvent `json:"recent_activity"`
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats := AdminStatsResponse{
		RecentActivity: []domain.AnalyticsEvent{},
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

	// 3. Premium Users (Assume 'FREE' is the default)
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE subscription_tier != 'FREE'").Scan(&stats.PremiumUsers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count premium users"})
		return
	}

	// 4. Events Last 24h & Recent Activity
	// Check if table exists first (optional, but good for safety if migration pending)
	// For now assume it exists based on previous sprints.
	var eventCount int
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_analytics_events WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&eventCount); err == nil {
		stats.EventsLast24h = eventCount
	}

	// 5. Trips Created 24h
	if err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM trips WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.TripsCreated24h); err != nil {
		// Ignore error
	}

	// 6. Recent Activity
	rows, err := h.DB.QueryContext(ctx, "SELECT id, user_id, event_type, created_at FROM user_analytics_events ORDER BY created_at DESC LIMIT 10")
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
