package handlers

import (
	"bytes"
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

func TestSaveTrip_QuotaEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Free User Duration Gate (> 3 days blocked)", func(t *testing.T) {
		mockSub := &MockSubService{}
		mockTrip := &MockTripService{}

		handler := NewTripHandler(mockTrip, mockSub, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := domain.Trip{
			Destination: "Bali",
			UserID:      "free_user",
			TripDays:    4,
		}
		body, _ := json.Marshal(reqBody)
		c.Request, _ = http.NewRequest("POST", "/api/v1/trips", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateTripAsync(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp domain.APIError
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "free_tier_limit", resp.Code)
	})

	t.Run("Free User Under Duration Limit (<= 3 days allowed)", func(t *testing.T) {
		mockSub := &MockSubService{}
		mockTrip := &MockTripService{}

		handler := NewTripHandler(mockTrip, mockSub, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := domain.Trip{
			Destination: "Bali",
			UserID:      "free_user",
			TripDays:    3,
		}
		body, _ := json.Marshal(reqBody)
		c.Request, _ = http.NewRequest("POST", "/api/v1/trips", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateTripAsync(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
