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

// --- Mocks using Interfaces ---

type MockTripService struct {
	CountUserTripsFunc func(ctx context.Context, userID string) (int, error)
	SaveUserTripFunc   func(ctx context.Context, trip *domain.Trip) error
}

func (m *MockTripService) GenerateTripStream(ctx context.Context, req domain.Trip, eventChan chan string, doneChan chan bool) {
}
func (m *MockTripService) GenerateTripAsync(ctx context.Context, req domain.Trip) (*domain.Trip, error) {
	return nil, nil
}
func (m *MockTripService) GetTrip(ctx context.Context, id string) (*domain.TripAndPlan, error) {
	return nil, nil
}
func (m *MockTripService) GetUserTrips(ctx context.Context, userID string) ([]domain.Trip, error) {
	return nil, nil
}
func (m *MockTripService) SaveUserTrip(ctx context.Context, trip *domain.Trip) error {
	if m.SaveUserTripFunc != nil {
		return m.SaveUserTripFunc(ctx, trip)
	}
	return nil
}
func (m *MockTripService) DeleteUserTrip(ctx context.Context, tripID string, userID string) error {
	return nil
}
func (m *MockTripService) CountUserTrips(ctx context.Context, userID string) (int, error) {
	if m.CountUserTripsFunc != nil {
		return m.CountUserTripsFunc(ctx, userID)
	}
	return 0, nil
}
func (m *MockTripService) GetActivityAlternatives(ctx context.Context, dest, orig, loc string, tags []string) ([]domain.ActivityAlternative, error) {
	return nil, nil
}
func (m *MockTripService) GetPackingList(ctx context.Context, tripID string) ([]domain.PackingCategory, error) {
	return nil, nil
}
func (m *MockTripService) GetDestinationDiscovery(ctx context.Context, city string) (*domain.DiscoveryResponse, error) {
	return nil, nil
}
func (m *MockTripService) RefineTrip(ctx context.Context, tripID, instruction string) (*domain.TripPlan, error) {
	return nil, nil
}
func (m *MockTripService) ExportTripToPDF(ctx context.Context, tripID string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *MockTripService) EnrichActivity(ctx context.Context, tripID string, dayIdx, actIdx int) (*domain.Activity, error) {
	return nil, nil
}

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

		handler := NewTripHandler(mockTrip, mockSub)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

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

		handler := NewTripHandler(mockTrip, mockSub)

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
