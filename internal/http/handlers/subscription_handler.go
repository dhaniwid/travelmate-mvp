package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	Service ISubscriptionService
}

func NewSubscriptionHandler(s ISubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s}
}

// GetSubscription returns the current user's subscription status
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	// 1. Get User Identity from Context (set by AuthMiddleware)
	userID := c.GetString("userID")
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
	userID := c.GetString("userID")
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

// CreateCheckoutSession handles the creation of a Stripe Checkout Session
func (h *SubscriptionHandler) CreateCheckoutSession(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing user_id"})
		return
	}
	email := c.GetString("email")

	// Get Price ID from request body or use default for PRO
	type Request struct {
		PriceID string `json:"price_id"`
	}
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, use default PRO price ID from env or constant
		// For now, let's require it or use a hardcoded one for MVP
		// req.PriceID = "price_H5ggYJDqBoJLm" // Example
	}

	// TODO: Get PriceID from config/env based on Tier?
	// For MVP, if PriceID is empty, assume Monthly PRO
	priceID := req.PriceID
	if priceID == "" {
		// FALLBACK: Use a hardcoded test price ID or get from config
		// This should ideally be in config
		priceID = "price_1QorB0P098234098" // REPLACE WITH REAL STRIPE PRICE ID
		// c.JSON(http.StatusBadRequest, gin.H{"error": "price_id is required"})
		// return
	}

	url, err := h.Service.CreateCheckoutSession(userID, email, priceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
