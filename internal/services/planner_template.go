package services

import (
	"context"
	"travelmate/internal/domain"
)

type TemplatePlanner struct{}

func NewTemplatePlanner() *TemplatePlanner {
	return &TemplatePlanner{}
}

// GenerateOnlyItinerary: Dummy Itinerary
func (t *TemplatePlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error) {
	var itinerary []domain.ItineraryDay

	for i := 1; i <= trip.TripDays; i++ {
		day := domain.ItineraryDay{Day: i}
		if i == 1 {
			day.Title = "Arrival & Cultural Immersion"
			day.Activities = []domain.Activity{
				{Time: "14:00", Activity: "Check-in hotel", Type: "Logistics", PlaceName: "Hotel Area", Description: "Check in process"},
				{Time: "16:00", Activity: "Explore City Center", Type: "Sightseeing", PlaceName: "City Square", Description: "Walking tour around the old town"},
			}
		} else {
			day.Title = "Adventure & Nature"
			day.Activities = []domain.Activity{
				{Time: "09:00", Activity: "Visit Iconic Temple", Type: "Culture", PlaceName: "Grand Temple", Description: "Historical site visit"},
				{Time: "13:00", Activity: "Local Cuisine Lunch", Type: "Culinary", PlaceName: "Warung Legendaris", Description: "Try the signature dish"},
			}
		}
		itinerary = append(itinerary, day)
	}

	return itinerary, nil
}

// GenerateTransportAndStay: Dummy Logistics (UPDATED STRUCT)
func (t *TemplatePlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	return domain.TripPlan{
		TripID: trip.ID,

		// 1. Context
		LogisticsContext: &domain.LogisticsContext{
			DistanceKM:   150,
			RouteType:    "Inter-City", // Updated Field
			WarningAlert: "⚠️ TEMPLATE MODE: Showing dummy data for testing.",
		},

		// 2. Budget
		BudgetBreakdown: domain.BudgetBreakdown{
			Transport:     500000,
			Accommodation: 1500000,
			Food:          1000000,
			Tickets:       300000,
			Misc:          200000,
		},

		// 3. Transport (SESUAIKAN FIELD BARU)
		TransportOptions: []domain.TransportOption{
			{
				StrategyTag:          "CEPAT",
				Name:                 "Express Train (Template)",
				PriceTier:            "MED",
				TotalDurationDisplay: "2h 15m", // Ganti EstimatedTime
				Pros:                 "Fastest overland option directly to city center.",
				OperatorsHint:        "KAI (Argo Parahyangan)",       // Field Baru
				BookingQuery:         "tiket kereta jakarta bandung", // Field Baru
				HubDetails: domain.HubDetails{
					DepartureNode: "Gambir Station",
					ArrivalNode:   "Bandung Station",
				},
				Breakdown: domain.TransportBreakdown{
					FirstMile: "Grab to Gambir (30m)",
					MainLeg:   "Argo Parahyangan (2h 45m)",
					LastMile:  "Taxi to Hotel (20m)",
				},
			},
			{
				StrategyTag:          "HEMAT",
				Name:                 "Shuttle Bus (Template)",
				PriceTier:            "LOW",
				TotalDurationDisplay: "4h 00m",
				Pros:                 "Budget friendly, flexible departure times.",
				OperatorsHint:        "CitiTrans, DayTrans, Baraya",
				BookingQuery:         "travel shuttle jakarta bandung",
				HubDetails: domain.HubDetails{
					DepartureNode: "Pool SCBD",
					ArrivalNode:   "Pool Dipatiukur",
				},
				Breakdown: domain.TransportBreakdown{
					FirstMile: "Ojek to Pool (15m)",
					MainLeg:   "Shuttle via Tol (3h 30m)",
					LastMile:  "Angkot to Area (15m)",
				},
			},
		},

		// 4. Accommodation (SESUAIKAN FIELD BARU)
		// Pastikan nama field struct ini sama dengan di domain.TripPlan (StrategicAccommodation atau AccommodationOptions)
		AccommodationOptions: []domain.AccommodationOption{
			{
				Type:                 "Hotel",
				AreaName:             "Braga District",                               // Ganti LocationArea
				RecommendationReason: "Historical center, walkable to cafes.",        // Ganti LocationNote
				Vibe:                 "Vintage atmosphere with lively night scenes.", // Ganti Description
			},
			{
				Type:                 "Villa",
				AreaName:             "Dago Pakar",
				RecommendationReason: "Best for city lights view and fresh air.",
				Vibe:                 "Cool, quiet, and romantic.",
			},
		},
	}, nil
}

// GenerateAlternatives: Dummy Alternatives
func (t *TemplatePlanner) GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	return []domain.ActivityAlternative{
		{
			Activity:    "Visit Art Gallery (Template)",
			Type:        "Arts",
			PlaceName:   "Selasar Sunaryo",
			Description: "Contemporary art space with coffee shop.",
		},
		{
			Activity:    "Night Walk (Template)",
			Type:        "Leisure",
			PlaceName:   "Asia Afrika Street",
			Description: "Enjoy the cosplay street performers.",
		},
	}, nil
}

// GenerateDiscovery: Dummy Discovery (Untuk Fitur Baru)
func (t *TemplatePlanner) GetDiscoveryInfo(ctx context.Context, city string) (*domain.DiscoveryResponse, error) {
	return &domain.DiscoveryResponse{
		City:    city + " (Template)",
		Tagline: "The City of Flowers and Fire.",
		Vibes:   []string{"Creative", "Culinary", "Cool Weather"},
		Highlights: []domain.PlaceHighlight{
			{Name: "Template Crater", Type: "Nature", Hook: "Active volcano accessible by car."},
			{Name: "Template Street", Type: "Urban", Hook: "Colonial architecture walk."},
		},
		CulinarySignature: []domain.CulinarySignature{
			{Name: "Template Noodle", Description: "Spicy noodles.", Tip: "Level 5 is deadly."},
		},
		HiddenGem: domain.HiddenGem{
			Name: "Secret Forest", Description: "Pine forest hidden in the north.",
		},
		HistorySnippet: "This city hosted a famous conference in 1955.",
	}, nil
}

// GeneratePackingList: Dummy Packing List
func (t *TemplatePlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error) {
	// Return data statis yang umum
	return []domain.PackingItem{
		{
			Category: "Essentials 📄",
			Items: []string{
				"Passport / ID Card",
				"Wallet & Cash (IDR)",
				"Travel Insurance Doc",
				"Booking Confirmations",
			},
		},
		{
			Category: "Clothing 👕",
			Items: []string{
				"T-Shirts (Day wear)",
				"Comfortable Walking Shoes",
				"Underwear & Socks",
				"Light Jacket / Hoodie",
				"Sleepwear",
			},
		},
		{
			Category: "Toiletries ,🛀",
			Items: []string{
				"Toothbrush & Toothpaste",
				"Shampoo & Body Wash (Travel size)",
				"Deodorant",
				"Sunscreen (SPF 50)",
				"Hand Sanitizer",
			},
		},
		{
			Category: "Gadgets 📱",
			Items: []string{
				"Smartphone & Charger",
				"Power Bank",
				"Headphones / Earbuds",
				"Universal Adapter",
			},
		},
		{
			Category: "Health & Meds 💊",
			Items: []string{
				"Personal Medication",
				"Vitamins",
				"Pain Killers (Paracetamol)",
				"Motion Sickness Pills",
			},
		},
	}, nil
}

// GeneratePlan: Full Trip Plan (Itinerary + Logistics)
func (t *TemplatePlanner) GeneratePlan(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	plan, err := t.GenerateTransportAndStay(ctx, trip)
	if err != nil {
		return domain.TripPlan{}, err
	}

	itinerary, err := t.GenerateOnlyItinerary(ctx, trip)
	if err != nil {
		return domain.TripPlan{}, err
	}

	plan.Itinerary = itinerary
	plan.PackingList, _ = t.GeneratePackingList(ctx, trip)

	return plan, nil
}
