package services

import (
	"context"
	"testing"
	"travelmate/internal/domain"
)

// --- Mocks ---

type MockUserRepo struct {
	GetUserByClerkIDFunc   func(ctx context.Context, id string) (*domain.User, error)
	UpsertUserFunc         func(ctx context.Context, user *domain.User) error
	UpdateSubscriptionFunc func(ctx context.Context, userID, tier, status, stripeCustID, stripeSubID string) error
	GetUserByStripeIDFunc  func(ctx context.Context, stripeCustID string) (*domain.User, error)
}

func (m *MockUserRepo) GetUserByClerkID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetUserByClerkIDFunc != nil {
		return m.GetUserByClerkIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockUserRepo) UpsertUser(ctx context.Context, user *domain.User) error {
	if m.UpsertUserFunc != nil {
		return m.UpsertUserFunc(ctx, user)
	}
	return nil
}
func (m *MockUserRepo) UpdateSubscription(ctx context.Context, userID, tier, status, stripeCustID, stripeSubID string) error {
	if m.UpdateSubscriptionFunc != nil {
		return m.UpdateSubscriptionFunc(ctx, userID, tier, status, stripeCustID, stripeSubID)
	}
	return nil
}
func (m *MockUserRepo) GetUserByStripeID(ctx context.Context, stripeCustID string) (*domain.User, error) {
	if m.GetUserByStripeIDFunc != nil {
		return m.GetUserByStripeIDFunc(ctx, stripeCustID)
	}
	return nil, nil
}

type MockSubRepo struct {
	GetQuotaFunc             func(ctx context.Context, userID, month string) (*domain.TripQuota, error)
	IncrementQuotaFunc       func(ctx context.Context, userID, month string) error
	LogSubscriptionEventFunc func(ctx context.Context, event *domain.SubscriptionEvent) error
}

func (m *MockSubRepo) GetQuota(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
	if m.GetQuotaFunc != nil {
		return m.GetQuotaFunc(ctx, userID, month)
	}
	return &domain.TripQuota{}, nil
}
func (m *MockSubRepo) IncrementQuota(ctx context.Context, userID, month string) error {
	if m.IncrementQuotaFunc != nil {
		return m.IncrementQuotaFunc(ctx, userID, month)
	}
	return nil
}
func (m *MockSubRepo) LogSubscriptionEvent(ctx context.Context, event *domain.SubscriptionEvent) error {
	if m.LogSubscriptionEventFunc != nil {
		return m.LogSubscriptionEventFunc(ctx, event)
	}
	return nil
}

type MockPaymentGateway struct {
	CreateCheckoutSessionFunc func(userID, email, priceID string) (string, error)
}

func (m *MockPaymentGateway) CreateCheckoutSession(userID, email, priceID string) (string, error) {
	if m.CreateCheckoutSessionFunc != nil {
		return m.CreateCheckoutSessionFunc(userID, email, priceID)
	}
	return "http://mock-checkout-url", nil
}

// --- Tests ---

func TestGetUserSubscription_ExistingUser(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{UserID: id, SubscriptionTier: "PRO", SubscriptionStatus: "ACTIVE"}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, &MockSubRepo{}, &MockPaymentGateway{})

	user, err := service.GetUserSubscription(context.Background(), "user_123", "test@example.com", "Test User")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if user.UserID != "user_123" {
		t.Errorf("Expected user_123, got %s", user.UserID)
	}
	if user.SubscriptionTier != "PRO" {
		t.Errorf("Expected PRO, got %s", user.SubscriptionTier)
	}
}

func TestGetUserSubscription_LazyCreation(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, nil // User not found
		},
		UpsertUserFunc: func(ctx context.Context, user *domain.User) error {
			if user.UserID != "user_new" {
				t.Errorf("Upserted wrong user ID: %s", user.UserID)
			}
			return nil
		},
	}
	service := NewSubscriptionService(mockRepo, &MockSubRepo{}, &MockPaymentGateway{})

	user, err := service.GetUserSubscription(context.Background(), "user_new", "new@example.com", "New User")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if user.UserID != "user_new" {
		t.Errorf("Expected user_new, got %s", user.UserID)
	}
	if user.SubscriptionTier != "FREE" {
		t.Errorf("Expected default FREE tier, got %s", user.SubscriptionTier)
	}
}

func TestGetUserQuota_ProUser(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{UserID: id, SubscriptionTier: "PRO", SubscriptionStatus: "ACTIVE"}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, &MockSubRepo{}, &MockPaymentGateway{})

	quota, err := service.GetUserQuota(context.Background(), "user_pro", "pro@example.com")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !quota.IsUnlimited {
		t.Error("Expected IsUnlimited to be true for PRO user")
	}
	if quota.Remaining != 9999 {
		t.Errorf("Expected virtual remaining 9999, got %d", quota.Remaining)
	}
}

func TestGetUserQuota_FreeUser_WithLimit(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{UserID: id, SubscriptionTier: "FREE", SubscriptionStatus: "ACTIVE"}, nil
		},
	}
	mockSubRepo := &MockSubRepo{
		GetQuotaFunc: func(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
			return &domain.TripQuota{
				UserID:       userID,
				Month:        month,
				TripsCreated: 1,
				QuotaLimit:   3,
			}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, mockSubRepo, &MockPaymentGateway{})

	quota, err := service.GetUserQuota(context.Background(), "user_free", "free@example.com")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if quota.IsUnlimited {
		t.Error("Expected IsUnlimited to be false for FREE user")
	}
	if quota.Remaining != 2 {
		t.Errorf("Expected remaining 2, got %d", quota.Remaining)
	}
}

func TestCheckQuotaAvailability_Blocked(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{UserID: id, SubscriptionTier: "FREE"}, nil
		},
	}
	mockSubRepo := &MockSubRepo{
		GetQuotaFunc: func(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
			return &domain.TripQuota{
				UserID:       userID,
				TripsCreated: 3,
				QuotaLimit:   3,
			}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, mockSubRepo, &MockPaymentGateway{})

	allowed, err := service.CheckQuotaAvailability(context.Background(), "user_blocked")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if allowed {
		t.Error("Expected allowed to be false when quota exceeded")
	}
}

// Test: Bonus Quota Integration (Referral System)
func TestCheckQuotaAvailability_WithBonusQuota(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				UserID:           id,
				SubscriptionTier: "FREE",
				BonusTripQuota:   2, // 🎁 User earned 2 bonus trips from referrals
			}, nil
		},
	}
	mockSubRepo := &MockSubRepo{
		GetQuotaFunc: func(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
			return &domain.TripQuota{
				UserID:       userID,
				TripsCreated: 4, // User has created 4 trips
				QuotaLimit:   3, // Base limit is 3
			}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, mockSubRepo, &MockPaymentGateway{})

	// Effective limit should be 3 (base) + 2 (bonus) = 5
	// User has created 4, so they should still be allowed
	allowed, err := service.CheckQuotaAvailability(context.Background(), "user_with_bonus")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !allowed {
		t.Error("Expected allowed to be true (4 trips < 5 effective limit)")
	}
}

// Test: Bonus Quota in GetUserQuota
func TestGetUserQuota_WithBonusQuota(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				UserID:           id,
				SubscriptionTier: "FREE",
				BonusTripQuota:   3, // 🎁 User earned 3 bonus trips
			}, nil
		},
	}
	mockSubRepo := &MockSubRepo{
		GetQuotaFunc: func(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
			return &domain.TripQuota{
				UserID:       userID,
				Month:        month,
				TripsCreated: 2,
				QuotaLimit:   3, // Base limit
			}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, mockSubRepo, &MockPaymentGateway{})

	quota, err := service.GetUserQuota(context.Background(), "user_bonus", "bonus@example.com")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Effective limit = 3 (base) + 3 (bonus) = 6
	// Remaining = 6 - 2 = 4
	if quota.Remaining != 4 {
		t.Errorf("Expected remaining 4 (6 effective - 2 created), got %d", quota.Remaining)
	}
}

// Test: Bonus Quota Blocks When Exceeded
func TestCheckQuotaAvailability_BonusQuotaExceeded(t *testing.T) {
	mockRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				UserID:           id,
				SubscriptionTier: "FREE",
				BonusTripQuota:   2, // Bonus quota
			}, nil
		},
	}
	mockSubRepo := &MockSubRepo{
		GetQuotaFunc: func(ctx context.Context, userID, month string) (*domain.TripQuota, error) {
			return &domain.TripQuota{
				UserID:       userID,
				TripsCreated: 5, // User has created 5 trips
				QuotaLimit:   3, // Base limit is 3
			}, nil
		},
	}
	service := NewSubscriptionService(mockRepo, mockSubRepo, &MockPaymentGateway{})

	// Effective limit = 3 + 2 = 5
	// User has created 5, so they should be blocked
	allowed, err := service.CheckQuotaAvailability(context.Background(), "user_exceeded")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if allowed {
		t.Error("Expected allowed to be false when effective quota exceeded (5 >= 5)")
	}
}
