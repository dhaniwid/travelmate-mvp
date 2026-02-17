package handlers

import (
	"bytes"
	"context"
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

	t.Run("Free User At Limit", func(t *testing.T) {
		// Mock Services
		mockSub := &MockSubService{
			GetUserQuotaFunc: func(ctx context.Context, userID, email string) (*domain.TripQuota, error) {
				return &domain.TripQuota{UserID: userID, TripsCreated: 3, QuotaLimit: 3, IsUnlimited: false}, nil
			},
		}
		mockTrip := &MockTripService{
			CountUserTripsFunc: func(ctx context.Context, userID string) (int, error) {
				return 3, nil
			},
		}

		handler := NewTripHandler(mockTrip, mockSub, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", "free_user")

		reqBody := map[string]interface{}{
			"id":      "trip_4",
			"user_id": "free_user",
		}
		body, _ := json.Marshal(reqBody)
		c.Request, _ = http.NewRequest("POST", "/api/trips", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SaveTrip(c)

		// Assert
		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "quota_exceeded", resp["error"])
	})

	t.Run("Free User Under Limit", func(t *testing.T) {
		// Mock Services
		mockSub := &MockSubService{
			GetUserQuotaFunc: func(ctx context.Context, userID, email string) (*domain.TripQuota, error) {
				return &domain.TripQuota{UserID: userID, TripsCreated: 1, QuotaLimit: 3, IsUnlimited: false}, nil
			},
		}
		mockTrip := &MockTripService{
			CountUserTripsFunc: func(ctx context.Context, userID string) (int, error) {
				return 1, nil
			},
		}

		handler := NewTripHandler(mockTrip, mockSub, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"id":      "trip_2",
			"user_id": "free_user",
		}
		body, _ := json.Marshal(reqBody)
		c.Request, _ = http.NewRequest("POST", "/api/trips", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SaveTrip(c)

		// It should NOT be 403. It might be 500 or 200 depending on SaveUserTrip mock,
		// but 403 means quota check blocked it.
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})
}
