package handlers

import (
	"context"
	"travelmate/internal/domain"
)

// --- Shared Mocks for Handler Tests ---

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

type MockTripService struct {
	CountUserTripsFunc func(ctx context.Context, userID string) (int, error)
	SaveUserTripFunc   func(ctx context.Context, trip *domain.Trip) error
	GetTripFunc        func(ctx context.Context, id string) (*domain.TripAndPlan, error)
}

func (m *MockTripService) GenerateTripStream(ctx context.Context, req domain.Trip, eventChan chan string, doneChan chan bool) {
}

func (m *MockTripService) GenerateTripAsync(ctx context.Context, req domain.Trip) (*domain.Trip, error) {
	return &domain.Trip{ID: "test_trip_id"}, nil
}

func (m *MockTripService) GetTrip(ctx context.Context, id string) (*domain.TripAndPlan, error) {
	if m.GetTripFunc != nil {
		return m.GetTripFunc(ctx, id)
	}
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

func (m *MockTripService) GetActivityAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	return nil, nil
}

func (m *MockTripService) GetActivityAlternativesByIndex(ctx context.Context, tripID string, dayIdx, actIdx int, force bool) ([]domain.ActivityAlternative, error) {
	return nil, nil
}

func (m *MockTripService) SwapActivity(ctx context.Context, tripID string, dayIdx, actIdx int, alt domain.ActivityAlternative) error {
	return nil
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

func (m *MockTripService) AddActivity(ctx context.Context, tripID string, dayIdx int, title, time string, autoEnhance bool) (*domain.TripPlan, error) {
	return nil, nil
}

func (m *MockTripService) GetAddActivitySuggestions(ctx context.Context, tripID string, dayIdx int, timeStr string) ([]domain.ActivityAlternative, error) {
	return nil, nil
}

func (m *MockTripService) DeleteActivity(ctx context.Context, tripID string, dayIdx, actIdx int) (*domain.TripPlan, error) {
	return nil, nil
}
