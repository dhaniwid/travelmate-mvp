package services

import (
	"context"
	"travelmate/internal/domain"
)

type PlannerEngine interface {
	GeneratePlan(ctx context.Context, trip domain.Trip, transportOptions []domain.TransportOption) (domain.TripPlan, error)
	GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error)
	GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error)
	GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error)
	GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error)
}
