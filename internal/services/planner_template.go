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
	// Return Dummy/Template Data sesuai struktur Logistics Strategy baru
	return domain.TripPlan{
		TripID: trip.ID,

		// 1. New Context
		LogisticsContext: domain.LogisticsContext{
			DistanceKM:   150, // Dummy distance
			WarningAlert: "Standard route conditions applied for template plan.",
		},

		// 2. Budget (Estimasi Kasar)
		BudgetBreakdown: domain.BudgetBreakdown{
			Transport:     500000,
			Accommodation: 1500000,
			Food:          1000000,
			Tickets:       300000,
			Misc:          200000,
		},

		// 3. New Transport Strategy
		TransportOptions: []domain.TransportOption{
			{
				StrategyTag:   "CEPAT",
				Name:          "Express Route (Template)",
				PriceTier:     "HIGH",
				EstimatedTime: "2h 30m",
				Pros:          "Fastest option available in our database.",
				HubDetails: domain.HubDetails{
					DepartureNode: "Main Airport/Station",
					ArrivalNode:   "Destination Central Hub",
				},
				Breakdown: domain.TransportBreakdown{
					FirstMile: "Taxi to Departure Hub (45m)",
					MainLeg:   "Direct Flight/Train (1h 15m)",
					LastMile:  "Taxi to Accommodation (30m)",
				},
			},
			{
				StrategyTag:   "HEMAT",
				Name:          "Economy Route (Template)",
				PriceTier:     "LOW",
				EstimatedTime: "5h 00m",
				Pros:          "Best value for money.",
				HubDetails: domain.HubDetails{
					DepartureNode: "Bus Terminal",
					ArrivalNode:   "Destination Terminal",
				},
				Breakdown: domain.TransportBreakdown{
					FirstMile: "Local Transport to Terminal (1h)",
					MainLeg:   "Intercity Bus (3h 30m)",
					LastMile:  "Local Transport to Stay (30m)",
				},
			},
		},

		// 4. New Accommodation Strategy
		AccommodationOptions: []domain.AccommodationOption{
			{
				Type:         "Hotel",
				LocationArea: "City Center Area",
				LocationNote: "Strategic location near main attractions.",
				Description:  "Vibrant area with easy access to culinary spots and landmarks.",
			},
			{
				Type:         "Villa",
				LocationArea: "Scenic Highlands",
				LocationNote: "Best for relaxation and nature views.",
				Description:  "Quiet atmosphere surrounded by nature, perfect for unwinding.",
			},
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

// GenerateAlternatives GenerateAlternatives: Versi Mock/Dummy
func (t *TemplatePlanner) GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	// Return data statis/palsu
	return []domain.ActivityAlternative{
		{
			Activity:    "Walk around City Center (Template)",
			Type:        "Leisure",
			PlaceName:   "City Center",
			Description: "Enjoy the local vibe without spending money (Mock Alternative).",
		},
		{
			Activity:    "Visit Local Museum (Template)",
			Type:        "Cultural",
			PlaceName:   "National Museum",
			Description: "Learn history indoors (Mock Alternative).",
		},
	}, nil
}

// GeneratePackingList GeneratePackingList: Versi Mock/Dummy
func (t *TemplatePlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error) {
	// Return data statis/palsu
	return []domain.PackingItem{
		{
			Category: "Essentials (Mock)",
			Items:    []string{"Passport", "Wallet", "Phone Charger"},
		},
		{
			Category: "Clothing (Mock)",
			Items:    []string{"T-Shirts", "Jeans", "Comfortable Shoes"},
		},
	}, nil
}
