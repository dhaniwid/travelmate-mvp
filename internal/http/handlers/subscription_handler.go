package handlers

import (
	"net/http"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	Service *services.SubscriptionService
}

func NewSubscriptionHandler(s *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s}
}

// GetSubscription returns the current user's subscription status
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	// 1. Get User Identity from Context (set by AuthMiddleware)
	userID := c.GetString("user_id")
	email := c.GetString("email")
	name := c.GetString("name")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing user_id in context"})
		return
	}

	// 2. Call Service
	user, err := h.Service.GetUserSubscription(c.Request.Context(), userID, email, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Return Response (DTO mapping can be done here if needed, but returning domain struct is fine for now)
	c.JSON(http.StatusOK, user)
}

// GetQuota returns the user's trip creation quota
func (h *SubscriptionHandler) GetQuota(c *gin.Context) {
	userID := c.GetString("user_id")
	email := c.GetString("email")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing user_id in context"})
		return
	}

	quota, err := h.Service.GetUserQuota(c.Request.Context(), userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quota)
}
