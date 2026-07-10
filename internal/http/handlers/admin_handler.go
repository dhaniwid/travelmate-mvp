package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
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

// AdminUserResult is the response shape for user search.
type AdminUserResult struct {
	UserID                string     `json:"user_id"`
	Email                 string     `json:"email"`
	Name                  string     `json:"name"`
	SubscriptionTier      string     `json:"subscription_tier"`
	SubscriptionStatus    string     `json:"subscription_status"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at"`
	SubscriptionEndsAt    *time.Time `json:"subscription_ends_at"`
	CreatedAt             time.Time  `json:"created_at"`
}

// GetUsers handles GET /api/v1/admin/users?search={email}
func (h *AdminHandler) GetUsers(c *gin.Context) {
	ctx := c.Request.Context()
	search := c.Query("search")

	var rows *sql.Rows
	var err error
	if search != "" {
		rows, err = h.DB.QueryContext(ctx, `
			SELECT user_id, COALESCE(email,''), COALESCE(name,''),
			       COALESCE(subscription_tier,'FREE'), COALESCE(subscription_status,'ACTIVE'),
			       subscription_started_at, subscription_ends_at, created_at
			FROM users
			WHERE email ILIKE $1 OR name ILIKE $1
			ORDER BY created_at DESC LIMIT 20`,
			"%"+search+"%")
	} else {
		rows, err = h.DB.QueryContext(ctx, `
			SELECT user_id, COALESCE(email,''), COALESCE(name,''),
			       COALESCE(subscription_tier,'FREE'), COALESCE(subscription_status,'ACTIVE'),
			       subscription_started_at, subscription_ends_at, created_at
			FROM users
			ORDER BY created_at DESC LIMIT 20`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB query failed"})
		return
	}
	defer rows.Close()

	users := []AdminUserResult{}
	for rows.Next() {
		var u AdminUserResult
		if err := rows.Scan(
			&u.UserID, &u.Email, &u.Name,
			&u.SubscriptionTier, &u.SubscriptionStatus,
			&u.SubscriptionStartedAt, &u.SubscriptionEndsAt, &u.CreatedAt,
		); err == nil {
			users = append(users, u)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": users, "count": len(users)})
}

type SetSubscriptionRequest struct {
	Tier         string `json:"tier" binding:"required"`
	DurationDays int    `json:"duration_days"`
}

// SetSubscription handles POST /api/v1/admin/users/:userId/subscription
func (h *AdminHandler) SetSubscription(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("userId")

	var req SetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Tier != "PRO" && req.Tier != "FREE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tier must be PRO or FREE"})
		return
	}

	now := time.Now()
	var startedAt *time.Time
	var endsAt *time.Time
	tier := req.Tier
	status := "ACTIVE"

	if tier == "PRO" {
		if req.DurationDays <= 0 {
			req.DurationDays = 30
		}
		start := now
		end := now.Add(time.Duration(req.DurationDays) * 24 * time.Hour)
		startedAt = &start
		endsAt = &end
	}

	_, err := h.DB.ExecContext(ctx, `
		UPDATE users
		SET subscription_tier = $1,
		    subscription_status = $2,
		    subscription_started_at = $3,
		    subscription_ends_at = $4,
		    updated_at = NOW()
		WHERE user_id = $5`,
		tier, status, startedAt, endsAt, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("update failed: %v", err)})
		return
	}

	// Fetch email for log
	var email string
	_ = h.DB.QueryRowContext(ctx, "SELECT COALESCE(email,'') FROM users WHERE user_id = $1", userID).Scan(&email)

	logMsg := fmt.Sprintf("✅ [ADMIN] %s granted to %s (user_id=%s)", tier, email, userID)
	if tier == "PRO" {
		logMsg = fmt.Sprintf("✅ [ADMIN] PRO granted to %s (user_id=%s) — %d days — expires %s",
			email, userID, req.DurationDays, endsAt.Format("2006-01-02"))
	} else {
		logMsg = fmt.Sprintf("⬇️ [ADMIN] Downgraded %s to FREE (user_id=%s)", email, userID)
	}
	log.Println(logMsg)

	// Log to analytics_events
	eventData := fmt.Sprintf(`{"tier":"%s","duration_days":%d,"by":"admin"}`, tier, req.DurationDays)
	_, _ = h.DB.ExecContext(ctx, `
		INSERT INTO user_analytics_events (user_id, event_type, event_data)
		VALUES ($1, $2, $3::jsonb)`,
		userID, "admin_subscription_change", eventData,
	)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  logMsg,
		"user_id":  userID,
		"tier":     tier,
		"ends_at":  endsAt,
	})
}
