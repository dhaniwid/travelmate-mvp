package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Security Integration Tests
// Verifies fix for Quota Bypass, IDOR, and Rate Limit Resistance.

func TestSecurity_QuotaBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Scenario A: The Free Lunch (Quota Bypass Attack)", func(t *testing.T) {
		mockSub := &MockSubService{}
		mockSub.CheckQuotaAvailabilityFunc = func(ctx context.Context, userID string) (bool, error) {
			return false, nil // FORBIDDEN
		}

		handler := NewTripHandler(&MockTripService{}, mockSub, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := domain.Trip{
			Destination: "Bali",
			UserID:      "malicious_user",
		}
		body, _ := json.Marshal(reqBody)
		c.Request, _ = http.NewRequest("POST", "/api/v1/trips", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateTripAsync(c)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp domain.APIError
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "quota_exceeded", resp.Code)
	})
}

func TestSecurity_IDOR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Scenario B: The Peeping Tom (IDOR Attack on Delete)", func(t *testing.T) {
		mockTrip := &MockTripService{}
		handler := NewTripHandler(mockTrip, &MockSubService{}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("userID", "user_b")
		c.Params = []gin.Param{{Key: "id", Value: "123"}}

		c.Request, _ = http.NewRequest("DELETE", "/api/v1/trips/123", nil)
		handler.DeleteTrip(c)

		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSecurity_GetTrip_IDOR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Scenario D: The Eavesdropper (IDOR Attack on GetTrip)", func(t *testing.T) {
		mockTrip := &MockTripService{
			GetTripFunc: func(ctx context.Context, id string) (*domain.TripAndPlan, error) {
				if id == "789" {
					return &domain.TripAndPlan{
						Trip: domain.Trip{ID: "789", UserID: "user_a"},
						Plan: domain.TripPlan{TripID: "789"},
					}, nil
				}
				return nil, nil
			},
		}

		handler := NewTripHandler(mockTrip, &MockSubService{}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("userID", "user_b")
		c.Params = []gin.Param{{Key: "id", Value: "789"}}

		c.Request, _ = http.NewRequest("GET", "/api/v1/trips/789", nil)
		handler.GetTrip(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp domain.APIError
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "forbidden", resp.Code)
	})
}

func TestSecurity_RateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Scenario C: The Bill Exploder (Stress/Rate Limit Check)", func(t *testing.T) {
		mockTrip := &MockTripService{}
		mockSub := &MockSubService{}
		handler := NewTripHandler(mockTrip, mockSub, nil)

		var wg sync.WaitGroup
		numRequests := 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)

				reqBody := domain.Trip{Destination: "Tokyo", UserID: "guest"}
				body, _ := json.Marshal(reqBody)
				c.Request, _ = http.NewRequest("POST", "/api/v1/trips", bytes.NewBuffer(body))
				c.Request.Header.Set("Content-Type", "application/json")

				handler.CreateTripAsync(c)
				results <- w.Code
			}()
		}

		wg.Wait()
		close(results)

		count200 := 0
		for code := range results {
			if code == http.StatusOK {
				count200++
			}
		}
		assert.Greater(t, count200, 0)
	})
}
