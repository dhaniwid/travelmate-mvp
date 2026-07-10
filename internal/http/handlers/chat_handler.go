package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
)

const chatDailyLimit = 5

// ChatHandler handles context-aware AI chat requests.
type ChatHandler struct {
	ChatService IChatService
	SubService  ISubscriptionService
	DB          *sql.DB
}

func NewChatHandler(chatService IChatService, subService ISubscriptionService, db *sql.DB) *ChatHandler {
	return &ChatHandler{ChatService: chatService, SubService: subService, DB: db}
}

// countChatMessages returns how many messages the user sent in the last 24h.
func (h *ChatHandler) countChatMessages(ctx context.Context, userID string) (int, error) {
	var count int
	err := h.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_messages WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`,
		userID,
	).Scan(&count)
	return count, err
}

// insertChatMessage records a sent message for rate-limit tracking.
func (h *ChatHandler) insertChatMessage(ctx context.Context, userID, tripID string) {
	if _, err := h.DB.ExecContext(ctx,
		`INSERT INTO chat_messages (user_id, trip_id) VALUES ($1, $2)`,
		userID, tripID,
	); err != nil {
		log.Printf("chat_messages insert error: %v", err)
	}
}

// GetChatUsage handles GET /api/v1/chat/usage
// Returns { used, limit, is_pro } so the frontend can render the counter.
func (h *ChatHandler) GetChatUsage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	isPro, err := h.SubService.IsPROUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("IsPROUser error in chat usage: %v", err)
		isPro = false
	}

	used := 0
	if !isPro {
		used, err = h.countChatMessages(c.Request.Context(), userID)
		if err != nil {
			log.Printf("countChatMessages error: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"used":   used,
		"limit":  chatDailyLimit,
		"is_pro": isPro,
	})
}

// ChatCompletion handles POST /api/v1/chat/completion
func (h *ChatHandler) ChatCompletion(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(string)

	var req struct {
		TripID  string `json:"trip_id" binding:"required"`
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trip_id and message are required"})
		return
	}

	// Rate limit check for FREE users
	isPro, err := h.SubService.IsPROUser(c.Request.Context(), uid)
	if err != nil {
		log.Printf("IsPROUser error in chat: %v", err)
		isPro = false
	}
	if !isPro {
		used, err := h.countChatMessages(c.Request.Context(), uid)
		if err != nil {
			log.Printf("countChatMessages error: %v", err)
		}
		if used >= chatDailyLimit {
			c.JSON(http.StatusTooManyRequests, domain.APIError{
				Code:    "chat_limit_reached",
				Message: "Kamu sudah pakai 5 pesan hari ini. Upgrade PRO untuk chat unlimited.",
			})
			return
		}
	}

	reply, err := h.ChatService.ChatWithTrip(c.Request.Context(), req.TripID, uid, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reply. Please try again."})
		return
	}

	// Record usage (async — don't block the response)
	if !isPro {
		go h.insertChatMessage(context.Background(), uid, req.TripID)
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}
