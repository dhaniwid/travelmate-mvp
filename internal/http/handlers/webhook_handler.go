package handlers

import (
	"io"
	"net/http"
	"travelmate/internal/services"
	stripePkg "travelmate/internal/stripe"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	Service      *services.SubscriptionService
	StripeClient *stripePkg.Client
}

func NewWebhookHandler(s *services.SubscriptionService, sc *stripePkg.Client) *WebhookHandler {
	return &WebhookHandler{Service: s, StripeClient: sc}
}

func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	event, err := h.StripeClient.ConstructEvent(payload, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error verifying webhook signature: " + err.Error()})
		return
	}

	if err := h.Service.HandleStripeEvent(c.Request.Context(), event); err != nil {
		// Log error but maybe return 200 so Stripe doesn't retry indefinitely for logic errors?
		// Better to return 500 so it retries transient errors.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error handling event: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
