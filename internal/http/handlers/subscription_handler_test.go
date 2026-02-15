package handlers

import (
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

type MockSubService struct {
	GetUserSubscriptionFunc    func(ctx context.Context, userID, email, name string) (*domain.User, error)
	GetUserQuotaFunc           func(ctx context.Context, userID, email string) (*domain.TripQuota, error)
	CheckQuotaAvailabilityFunc func(ctx context.Context, userID string) (bool, error)
}

func (m *MockSubService) GetUserSubscription(ctx context.Context, userID, email, name string) (*domain.User, error) {
	if m.GetUserSubscriptionFunc != nil {
		return m.GetUserSubscriptionFunc(ctx, userID, email, name)
	}
	if userID == "pro_user" {
		return &domain.User{UserID: "pro_user", SubscriptionTier: "PRO", SubscriptionStatus: "ACTIVE"}, nil
	}
	return &domain.User{UserID: userID, SubscriptionTier: "FREE", SubscriptionStatus: "ACTIVE"}, nil
}
func (m *MockSubService) GetUserQuota(ctx context.Context, userID, email string) (*domain.TripQuota, error) {
	if m.GetUserQuotaFunc != nil {
		return m.GetUserQuotaFunc(ctx, userID, email)
	}
	return &domain.TripQuota{UserID: userID, TripsCreated: 1, QuotaLimit: 3, Remaining: 2}, nil
}
func (m *MockSubService) CreateCheckoutSession(userID, email, priceID string) (string, error) {
	return "http://mock-checkout-url", nil
}
func (m *MockSubService) CheckQuotaAvailability(ctx context.Context, userID string) (bool, error) {
	if m.CheckQuotaAvailabilityFunc != nil {
		return m.CheckQuotaAvailabilityFunc(ctx, userID)
	}
	return true, nil
}
func (m *MockSubService) IncrementQuota(ctx context.Context, userID string) error {
	return nil
}

// --- Tests ---

func TestGetSubscription_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Free User", func(t *testing.T) {
		handler := NewSubscriptionHandler(&MockSubService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/subscription", nil)
		c.Set("user_id", "free_user")

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
		c.Set("user_id", "pro_user")

		handler.GetSubscription(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp domain.User
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "PRO", resp.SubscriptionTier)
	})
}
