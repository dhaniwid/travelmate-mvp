package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// --- Tests ---

// --- Tests ---

func TestGetSubscription_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Free User", func(t *testing.T) {
		handler := NewSubscriptionHandler(&MockSubService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/subscription", nil)
		c.Set("userID", "free_user")

		handler.GetSubscription(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp domain.User
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "FREE", resp.SubscriptionTier)
	})

	t.Run("Pro User", func(t *testing.T) {
		handler := NewSubscriptionHandler(&MockSubService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/subscription", nil)
		c.Set("userID", "pro_user")

		handler.GetSubscription(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp domain.User
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "PRO", resp.SubscriptionTier)
	})
}
