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
func (t *TemplatePlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
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

	return domain.ItineraryResponse{
		Itinerary:       itinerary,
		MorningBriefing: "Ready for your adventure in " + trip.Destination + "! Enjoy the local culture.",
		Highlights:      []domain.TripHighlight{},
	}, nil
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

// GenerateActivityReplacement: Dummy implementation
func (t *TemplatePlanner) GenerateActivityReplacement(ctx context.Context, dest, activity string, tags []string) ([]domain.ActivityAlternative, error) {
	return t.GenerateAlternatives(ctx, dest, activity, "", tags)
}

func (t *TemplatePlanner) EnhanceActivity(ctx context.Context, dest, title string) (*domain.Activity, error) {
	return &domain.Activity{
		Activity:    title,
		Description: "A beautiful place to visit (Template)",
		PlaceName:   dest,
		Type:        "Leisure",
		IsSkeleton:  false,
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
func (t *TemplatePlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingCategory, error) {
	// Return data statis yang umum
	return []domain.PackingCategory{
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
			Category: "Toiletries 🛀",
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

	resp, err := t.GenerateOnlyItinerary(ctx, trip)
	if err != nil {
		return domain.TripPlan{}, err
	}

	plan.Itinerary = resp.Itinerary
	plan.MorningBriefing = resp.MorningBriefing
	plan.Highlights = resp.Highlights
	plan.PackingList, _ = t.GeneratePackingList(ctx, trip)

	return plan, nil
}

// GenerateEditorial: Dummy Editorial Content
func (t *TemplatePlanner) GenerateEditorial(ctx context.Context, trip domain.Trip) (domain.EditorialResponse, error) {
	return domain.EditorialResponse{
		Tagline:         "The City of Flowers and Fire (Template)",
		Vibes:           []string{"Creative", "Culinary", "Cool Weather"},
		MorningBriefing: "Ready for your adventure in " + trip.Destination + "! Enjoy the local culture.",
		HistorySnippet:  "This city hosted a famous conference in 1955.",
		Highlights: []domain.TripHighlight{
			{Title: "Template Crater", Type: "Nature", Hook: "Active volcano accessible by car."},
			{Title: "Template Street", Type: "Urban", Hook: "Colonial architecture walk."},
		},
		CulinarySignature: []domain.CulinarySignature{
			{Name: "Template Noodle", Description: "Spicy noodles.", Tip: "Level 5 is deadly."},
		},
		HiddenGem: &domain.HiddenGem{
			Name: "Secret Forest", Description: "Pine forest hidden in the north.",
		},
	}, nil
}

// RefineItinerary: Template implementation (returns unchanged itinerary)
func (t *TemplatePlanner) RefineItinerary(ctx context.Context, currentItinerary []domain.ItineraryDay, instruction string) ([]domain.ItineraryDay, error) {
	// Template mode doesn't support refinement, return as-is
	return currentItinerary, nil
}

// GenerateUltraConciseItinerary (Stub)
func (t *TemplatePlanner) GenerateUltraConciseItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	// Re-use logic from GenerateOnlyItinerary
	return t.GenerateOnlyItinerary(ctx, trip)
}

// GenerateEnrichmentDetails (Stub)
func (t *TemplatePlanner) GenerateEnrichmentDetails(ctx context.Context, skeleton domain.TripPlan) (domain.TripPlan, error) {
	// Just return the skeleton combined with dummy logistics
	// In real template mode, we'd probably just return the full template plan immediately
	// For now, let's just return what we have plus some dummy data to simulate enrichment
	return t.GenerateTransportAndStay(ctx, domain.Trip{ID: skeleton.TripID})
}

// GenerateFullItineraryPass (Stub)
func (t *TemplatePlanner) GenerateFullItineraryPass(ctx context.Context, trip domain.Trip) (domain.AIPlannerResponse, error) {
	// Re-use logic from GeneratePlan but map to AIPlannerResponse
	plan, _ := t.GeneratePlan(ctx, trip)
	return domain.AIPlannerResponse{
		Itinerary:            plan.Itinerary,
		BudgetBreakdown:      plan.BudgetBreakdown,
		TransportOptions:     plan.TransportOptions,
		AccommodationOptions: plan.AccommodationOptions,
		PackingList:          plan.PackingList,
		MorningBriefing:      plan.MorningBriefing,
		Highlights:           plan.Highlights,
	}, nil
}

// GenerateTripCore (Stage 1 Stub)
func (t *TemplatePlanner) GenerateTripCore(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	return t.GenerateOnlyItinerary(ctx, trip)
}

// EnrichTripVibe (Stage 2 Stub)
func (t *TemplatePlanner) EnrichTripVibe(ctx context.Context, stage1JSON string) (domain.TripVibeResponse, error) {
	return domain.TripVibeResponse{
		ItineraryUpdates: []struct {
			Day           int    `json:"day"`
			ActivityIndex int    `json:"activity_index"`
			Description   string `json:"description"`
			VisitDuration string `json:"visit_duration"`
			Category      string `json:"category"`
		}{},
		Highlights: []domain.TripHighlight{
			{Title: "Template Highlight", Type: "Sightseeing", Hook: "Must see."},
		},
	}, nil
}

// GenerateTripLogistics (Stage 3 Stub)
func (t *TemplatePlanner) GenerateTripLogistics(ctx context.Context, trip domain.Trip) (domain.TripLogisticsResponse, error) {
	plan, _ := t.GenerateTransportAndStay(ctx, trip)
	return domain.TripLogisticsResponse{
		ArrivalGuide:           domain.ArrivalGuide{},
		BudgetBreakdown:        plan.BudgetBreakdown,
		PackingList:            plan.PackingList,
		StrategicAccommodation: plan.AccommodationOptions,
	}, nil
}

// GenerateTripOverview (Stub)
func (t *TemplatePlanner) GenerateTripOverview(ctx context.Context, trip domain.Trip) (domain.TripOverviewResponse, error) {
	return domain.TripOverviewResponse{
		TripTitle:       "Tropical Escape to " + trip.Destination,
		MorningBriefing: "Pack your sunscreen and get ready for adventure!",
		ArrivalGuide: domain.ArrivalGuide{
			PrimaryTransport:    "Plane via Template Airways",
			TravelTime:          "4h 20m",
			EstimatedPriceRange: "$300 - $500",
			VisaInfo:            "Not required for most visitors.",
			BestTimeVisit:       "May to September",
		},
		BudgetBreakdown: domain.BudgetBreakdown{
			Transport:     200,
			Accommodation: 500,
			Food:          300,
			Tickets:       150,
			Misc:          50,
		},
		Highlights: []domain.TripHighlight{
			{Title: "Crystal Beach", Type: "Relaxing", Hook: "White sands and turquoise waters."},
			{Title: "Ancient Ruins", Type: "Sightseeing", Hook: "Explore the remnants of a lost civilization."},
		},
		StrategicAccommodation: []domain.AccommodationOption{
			{
				Type:                 "Resort",
				AreaName:             "Sunrise Bay",
				RecommendationReason: "Direct beach access and premium amenities.",
				Vibe:                 "Luxury & Tranquility",
				HotelSuggestions:     []string{"Template Beach Resort", "Azure Bay Hotel"},
			},
		},
	}, nil
}

// GenerateTripItinerary (Stub)
func (t *TemplatePlanner) GenerateTripItinerary(ctx context.Context, trip domain.Trip, overviewJSON string) (domain.ItineraryResponse, error) {
	return t.GenerateOnlyItinerary(ctx, trip)
}

// GenerateAddActivitySuggestions (Stub)
func (t *TemplatePlanner) GenerateAddActivitySuggestions(ctx context.Context, destination, style, bucket, time string) ([]domain.ActivityAlternative, error) {
	return []domain.ActivityAlternative{
		{Activity: "Morning Market Visit (Template)", Type: "Culinary"},
		{Activity: "Temple Photo Op (Template)", Type: "Sightseeing"},
		{Activity: "Hidden Alley Cafe (Template)", Type: "Leisure"},
	}, nil
}

// GetRegeneratePrompt (Stub)
func (t *TemplatePlanner) GetRegeneratePrompt(ctx context.Context, trip domain.Trip, prefs domain.UserPreferences) (string, error) {
	return "TEMPLATE MODE: No dynamic AI prompt generated.", nil
}

// GenerateTripSkeleton (Stub)
func (t *TemplatePlanner) GenerateTripSkeleton(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	// Re-use logic from GenerateOnlyItinerary
	return t.GenerateOnlyItinerary(ctx, trip)
}

func (t *TemplatePlanner) GenerateSkeletonStreaming(ctx context.Context, trip domain.Trip, ragContext string) (domain.ItineraryResponse, error) {
	return t.GenerateOnlyItinerary(ctx, trip)
}

func (t *TemplatePlanner) FetchRAGContext(_ context.Context, _, _ string, _ int) string {
	return ""
}

func (t *TemplatePlanner) GenerateTransportOnDemand(_ context.Context, _, _ string, _ int) ([]domain.TransportOption, error) {
	return []domain.TransportOption{}, nil
}
