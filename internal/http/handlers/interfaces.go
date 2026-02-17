package handlers

import (
	"context"
	"travelmate/internal/domain"
)

type ITripService interface {
	GenerateTripStream(ctx context.Context, req domain.Trip, eventChan chan string, doneChan chan bool)
	GenerateTripAsync(ctx context.Context, req domain.Trip) (*domain.Trip, error)
	GetTrip(ctx context.Context, id string) (*domain.TripAndPlan, error)
	GetUserTrips(ctx context.Context, userID string) ([]domain.Trip, error)
	SaveUserTrip(ctx context.Context, trip *domain.Trip) error
	DeleteUserTrip(ctx context.Context, tripID string, userID string) error
	CountUserTrips(ctx context.Context, userID string) (int, error)
	GetActivityAlternatives(ctx context.Context, dest, orig, loc string, tags []string) ([]domain.ActivityAlternative, error)
	GetActivityAlternativesByIndex(ctx context.Context, tripID string, dayIdx, actIdx int, force bool) ([]domain.ActivityAlternative, error)
	SwapActivity(ctx context.Context, tripID string, dayIdx, actIdx int, alt domain.ActivityAlternative) error
	GetPackingList(ctx context.Context, tripID string) ([]domain.PackingCategory, error)
	GetDestinationDiscovery(ctx context.Context, city string) (*domain.DiscoveryResponse, error)
	RefineTrip(ctx context.Context, tripID, instruction string) (*domain.TripPlan, error)
	ExportTripToPDF(ctx context.Context, tripID string) ([]byte, string, error)
	EnrichActivity(ctx context.Context, tripID string, dayIdx, actIdx int) (*domain.Activity, error)
	AddActivity(ctx context.Context, tripID string, dayIdx int, title, time string, autoEnhance bool) (*domain.TripPlan, error)
	GetAddActivitySuggestions(ctx context.Context, tripID string, dayIdx int, timeStr string) ([]domain.ActivityAlternative, error)
	DeleteActivity(ctx context.Context, tripID string, dayIdx, actIdx int) (*domain.TripPlan, error)
}

type ISubscriptionService interface {
	GetUserSubscription(ctx context.Context, userID, email, name string) (*domain.User, error)
	GetUserQuota(ctx context.Context, userID, email string) (*domain.TripQuota, error)
	CreateCheckoutSession(userID, email, priceID string) (string, error)
	CheckQuotaAvailability(ctx context.Context, userID string) (bool, error)
	IncrementQuota(ctx context.Context, userID string) error
}

type IReferralService interface {
	ProcessReferral(ctx context.Context, newUserID, referralCode string) error
	GetReferralStats(ctx context.Context, userID string) (*domain.ReferralStats, error)
}

type ICollaboratorRepository interface {
	HasAccess(ctx context.Context, tripID, userID string) (bool, error)
}
