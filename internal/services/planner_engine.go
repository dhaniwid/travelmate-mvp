package services

import (
	"context"
	"travelmate/internal/domain"
)

type PlannerEngine interface {
	GetDiscoveryInfo(ctx context.Context, city string) (*domain.DiscoveryResponse, error)
	GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error)
	GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error)
	GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error)
	GenerateActivityReplacement(ctx context.Context, dest, activity string, tags []string) ([]domain.ActivityAlternative, error)
	EnhanceActivity(ctx context.Context, dest, title string) (*domain.Activity, error)
	GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingCategory, error)
	GeneratePlan(ctx context.Context, trip domain.Trip) (domain.TripPlan, error)
	GenerateEditorial(ctx context.Context, trip domain.Trip) (domain.EditorialResponse, error)
	RefineItinerary(ctx context.Context, currentItinerary []domain.ItineraryDay, instruction string) ([]domain.ItineraryDay, error)
	GenerateUltraConciseItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error)
	GenerateEnrichmentDetails(ctx context.Context, skeleton domain.TripPlan) (domain.TripPlan, error)
	GenerateFullItineraryPass(ctx context.Context, trip domain.Trip) (domain.AIPlannerResponse, error)
	GenerateTripCore(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error)
	EnrichTripVibe(ctx context.Context, stage1JSON string) (domain.TripVibeResponse, error)
	GenerateTripSkeleton(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error)
	GenerateTripLogistics(ctx context.Context, trip domain.Trip) (domain.TripLogisticsResponse, error)
	GenerateTripOverview(ctx context.Context, trip domain.Trip) (domain.TripOverviewResponse, error)
	GenerateTripItinerary(ctx context.Context, trip domain.Trip, overviewJSON string) (domain.ItineraryResponse, error)
	GenerateAddActivitySuggestions(ctx context.Context, destination, style, bucket, time string) ([]domain.ActivityAlternative, error)
	GetRegeneratePrompt(ctx context.Context, trip domain.Trip, prefs domain.UserPreferences) (string, error)
}
