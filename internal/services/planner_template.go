package services

import (
	"context"
	"travelmate/internal/domain"
)

type TemplatePlanner struct{}

func NewTemplatePlanner() *TemplatePlanner {
	return &TemplatePlanner{}
}

func (t *TemplatePlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error) {
	var itinerary []domain.ItineraryDay

	for i := 1; i <= trip.TripDays; i++ {
		day := domain.ItineraryDay{Day: i}
		if i == 1 {
			day.Title = "Arrival & City Walk"
			day.Activities = []domain.Activity{
				{Time: "14:00", Activity: "Check-in hotel", Type: "Logistics", PlaceName: "Hotel", Description: "Check in and rest"},
				{Time: "16:00", Activity: "Visit landmark", Type: "Sightseeing", PlaceName: "City Icon", Description: "Light walking"},
			}
		} else {
			day.Title = "Exploration Day"
			day.Activities = []domain.Activity{
				{Time: "09:00", Activity: "Main Site Visit", Type: "Sightseeing", PlaceName: "Famous Spot", Description: "Must visit site"},
			}
		}
		itinerary = append(itinerary, day)
	}

	return itinerary, nil
}

func (t *TemplatePlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	return domain.TripPlan{
		TripID: trip.ID,
		BudgetBreakdown: domain.BudgetBreakdown{
			Transport:     1500000,
			Accommodation: 2000000,
			Food:          1000000,
		},
		TransportOptions: []domain.TransportOption{
			{Type: "Flight", Name: "Template Air", Price: 1200000, EstimatedTime: "1h 30m", Pros: "Fastest"},
		},
		AccommodationOptions: []domain.AccommodationOption{
			{Name: "Template Central Hotel", Type: "Hotel", Rating: "4.0", PricePerNight: 500000},
		},
	}, nil
}

func (t *TemplatePlanner) GeneratePlan(ctx context.Context, trip domain.Trip, transportOptions []domain.TransportOption) (domain.TripPlan, error) {
	itinerary, _ := t.GenerateOnlyItinerary(ctx, trip)
	plan, _ := t.GenerateTransportAndStay(ctx, trip)

	plan.Itinerary = itinerary
	if len(transportOptions) > 0 {
		plan.TransportOptions = transportOptions
	}

	return plan, nil
}
