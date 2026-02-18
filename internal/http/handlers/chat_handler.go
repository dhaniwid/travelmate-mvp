package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles context-aware AI chat requests.
type ChatHandler struct {
	ChatService IChatService
}

func NewChatHandler(chatService IChatService) *ChatHandler {
	return &ChatHandler{ChatService: chatService}
}

// ChatCompletion handles POST /api/v1/chat/completion
// Payload: { "trip_id": "...", "message": "..." }
// Response: { "reply": "..." }
func (h *ChatHandler) ChatCompletion(c *gin.Context) {
	// Extract authenticated user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		TripID  string `json:"trip_id" binding:"required"`
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trip_id and message are required"})
		return
	}

	reply, err := h.ChatService.ChatWithTrip(c.Request.Context(), req.TripID, userID.(string), req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reply. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}
