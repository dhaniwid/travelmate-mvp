package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	Service *services.SubscriptionService
}

func NewWebhookHandler(s *services.SubscriptionService) *WebhookHandler {
	return &WebhookHandler{Service: s}
}

// MayarWebhookPayload represents the Mayar.id webhook event body.
// Mayar.id sends a JSON body and signs it using HMAC-SHA256 with the secret key.
// The signature is sent in the X-Mayar-Signature header as a hex digest.
type MayarWebhookPayload struct {
	Event  string          `json:"event"`  // e.g. "payment.paid"
	Data   json.RawMessage `json:"data"`   // Event-specific payload
}

type MayarPaymentData struct {
	ID          string `json:"id"`
	ReferenceID string `json:"referenceId"` // user_id passed during checkout creation
	Email       string `json:"email"`
	Status      string `json:"status"` // "paid", "expired", "failed"
	Amount      int64  `json:"amount"`
}

// HandleMayarWebhook processes Mayar.id payment webhook events.
// Active payment provider as of Sprint 12.
func (h *WebhookHandler) HandleMayarWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return
	}

	// 1. Verify HMAC-SHA256 signature
	secret := os.Getenv("MAYAR_WEBHOOK_SECRET")
	if secret == "" {
		log.Println("[MayarWebhook] MAYAR_WEBHOOK_SECRET not set — rejecting request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Webhook secret not configured"})
		return
	}

	incomingSig := c.GetHeader("X-Mayar-Signature")
	if !verifyMayarSignature(payload, incomingSig, secret) {
		log.Printf("[MayarWebhook] Signature mismatch — possible spoofed request")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	// 2. Parse event
	var event MayarWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	log.Printf("[MayarWebhook] Received event: %s", event.Event)

	// 3. Handle event types
	switch event.Event {
	case "payment.paid":
		var data MayarPaymentData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment data"})
			return
		}
		if err := h.Service.HandleMayarPaymentPaid(c.Request.Context(), data.ReferenceID, data.Email, data.ID); err != nil {
			log.Printf("[MayarWebhook] Error handling payment.paid: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process payment"})
			return
		}

	default:
		// Unknown event — acknowledge receipt but take no action
		log.Printf("[MayarWebhook] Unhandled event type: %s", event.Event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func verifyMayarSignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
