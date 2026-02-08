package services

import (
	"context"
	"travelmate/internal/domain"
)

type PlannerEngine interface {
	GetDiscoveryInfo(ctx context.Context, city string) (*domain.DiscoveryResponse, error)
	GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error)
	GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error)
	GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error)
	GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error)
	GeneratePlan(ctx context.Context, trip domain.Trip) (domain.TripPlan, error)
}
