package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"

	"github.com/google/uuid"
)

type TripService struct {
	TripRepo       *repositories.TripRepository
	FeedbackRepo   *repositories.FeedbackRepository
	AccomRepo      *repositories.AccommodationRepository
	AttractionRepo *repositories.AttractionRepository
	TransportRepo  *repositories.TransportRepository
	PerfRepo       *repositories.PerformanceRepository
	Planner        PlannerEngine
	TransportServ  *TransportService
	LocationServ   *LocationService
	ImageSvc       *ImageService
}

func NewTripService(
	tr *repositories.TripRepository,
	fr *repositories.FeedbackRepository,
	ar *repositories.AccommodationRepository,
	attractionRepo *repositories.AttractionRepository,
	transRepo *repositories.TransportRepository,
	perfRepo *repositories.PerformanceRepository,
	p PlannerEngine,
	locS *LocationService,
	transportS *TransportService,
	imageSvc *ImageService,
) *TripService {
	return &TripService{
		TripRepo:       tr,
		FeedbackRepo:   fr,
		AccomRepo:      ar,
		AttractionRepo: attractionRepo,
		TransportRepo:  transRepo,
		PerfRepo:       perfRepo,
		Planner:        p,
		TransportServ:  transportS,
		LocationServ:   locS,
		ImageSvc:       imageSvc,
	}
}

// GenerateAndSaveTrip ==========================================================
// 1. LEGACY FUNCTION (Sequential & Blocking)
// ==========================================================
func (s *TripService) GenerateAndSaveTrip(ctx context.Context, req domain.Trip) (*domain.TripAndPlan, error) {
	startTime := time.Now()
	log.Printf("📜 [LEGACY] Starting sequential generation for: %s", req.Destination)

	// --- STEP A: TENTUKAN DESTINASI ---
	targetDestination := req.Destination
	isAutoDestination := false
	if targetDestination == "" {
		targetDestination = recommendDestination(req.Style)
		isAutoDestination = true
	}

	// --- STEP B: ENRICHMENT ---
	locData, err := s.LocationServ.GetOrEnrichLocation(ctx, targetDestination)
	if err == nil && locData != nil {
		req.LocationID = locData.ID
		req.Destination = locData.Name
	}

	// --- STEP C: TICKETS ---
	tickets, _ := s.TransportServ.SearchRealtimeTickets(ctx, req.Origin, req.Destination)

	// --- STEP D: PREPARE DATA ---
	req.ID = uuid.New().String()
	req.CreatedAt = time.Now()

	// --- STEP E: GENERATE PLAN (BLOCKING AI CALL) ---
	plan, err := s.Planner.GeneratePlan(ctx, req, tickets)
	if err != nil {
		return nil, err
	}
	plan.TripID = req.ID

	// Enrichment gambar (Sekuensial di versi legacy)
	//s.enrichWithImages(&plan)

	if isAutoDestination {
		plan.DecisionNotes = append(plan.DecisionNotes, fmt.Sprintf("✨ Destination chosen based on your '%s' style.", req.Style))
	}

	// --- STEP F: PERSISTENCE ---
	// Kita gunakan fungsi yang sama dengan versi stream agar konsisten
	err = s.TripRepo.SaveTripPlan(ctx, req, plan)
	if err != nil {
		return nil, err
	}

	// Background Mining
	go s.mineAttractions(context.Background(), req.LocationID, plan.Itinerary)
	go s.mineAccommodations(context.Background(), req.LocationID, plan.AccommodationOptions)

	// Log Performance
	s.PerfRepo.SaveMetric(ctx, "legacy_generate_full", time.Since(startTime), req.Destination, "gpt-4o")

	return &domain.TripAndPlan{Trip: req, Plan: plan}, nil
}

// GenerateTripStream ==========================================================
// 2. STREAM FUNCTION (Parallel & Responsive)
// ==========================================================
func (s *TripService) GenerateTripStream(ctx context.Context, trip domain.Trip, eventChan chan string, doneChan chan bool) {
	startTime := time.Now()
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// 1. DB LOOKUP (The 0-Second Win)
	print("🔍 Checking for existing plan in DB...\n")
	fmt.Printf("Trip Criteria - Dest: %s, Style: %s, Days: %d\n", trip.Destination, trip.Style, trip.TripDays)
	existingPlan, err := s.TripRepo.GetExistingPlanByCriteria(ctx, trip.Destination, trip.Style, trip.TripDays)
	fmt.Printf("Found existing plan: %v, err: %v\n", existingPlan != nil, err)
	if err == nil && existingPlan != nil {
		log.Printf("⚡ [FAST PATH] Found existing plan for %s, skipping AI", trip.Destination)
		s.sendEvent(eventChan, "metadata", map[string]string{"trip_id": trip.ID})
		s.sendEvent(eventChan, "itinerary", existingPlan.Itinerary)
		s.sendEvent(eventChan, "logistics", existingPlan)
		doneChan <- true
		log.Printf("🏁 [STREAM OPTIMIZED] UI is ready in: %v", time.Since(startTime))
		return
	}

	// 2. TENTUKAN LOKASI
	locData, _ := s.LocationServ.GetOrEnrichLocation(context.Background(), trip.Destination)
	if locData != nil {
		trip.LocationID = locData.ID
		trip.Destination = locData.Name
	}

	s.sendEvent(eventChan, "metadata", map[string]string{"trip_id": trip.ID})

	var wg sync.WaitGroup
	wg.Add(2)

	var mu sync.Mutex
	var finalItinerary []domain.ItineraryDay
	var finalPlan domain.TripPlan

	// TASK 1: Itinerary (Parallel)
	go func() {
		defer wg.Done()
		iti, _ := s.Planner.GenerateOnlyItinerary(ctx, trip)
		mu.Lock()
		finalItinerary = iti
		mu.Unlock()
		s.sendEvent(eventChan, "itinerary", iti)
	}()

	// TASK 2: Logistics & ASYNC Images
	go func() {
		defer wg.Done()
		plan, err := s.Planner.GenerateTransportAndStay(ctx, trip)
		if err != nil {
			log.Printf("❌ [TASK 2 FAILED] Logistics generation error: %v", err)
			return
		}
		if err == nil {
			s.sendEvent(eventChan, "logistics", plan)

			go func() {
				//s.enrichWithImages(&plan)
				//s.sendEvent(eventChan, "logistics_update", plan) // Update UI dengan gambar

				mu.Lock()
				finalPlan = plan
				mu.Unlock()
			}()
		}
	}()

	wg.Wait()

	finalPlan.TripID = trip.ID
	finalPlan.Itinerary = finalItinerary

	log.Printf("🏁 [STREAM OPTIMIZED] UI is ready in: %v", time.Since(startTime))

	go s.FinalizeAndSaveToDB(trip, finalPlan)
	doneChan <- true
}

// --- HELPERS & MINING FUNCTIONS ---

func (s *TripService) FinalizeAndSaveToDB(trip domain.Trip, plan domain.TripPlan) {
	ctx := context.Background()

	// 1. Simpan Trip Plan user
	_ = s.TripRepo.SaveTripPlan(ctx, trip, plan)

	// 2. Mining Accommodations (Seed DB)
	go s.mineAccommodations(ctx, trip.LocationID, plan.AccommodationOptions)

	// 3. Mining Attractions (Seed DB)
	go s.mineAttractions(ctx, trip.LocationID, plan.Itinerary)

	// 4. Mining Transports (Seed DB) - NEW!
	go s.mineTransports(ctx, trip.Origin, trip.Destination, plan.TransportOptions)
}

func (s *TripService) mineTransports(ctx context.Context, origin, dest string, transports []domain.TransportOption) {
	if origin == "" || dest == "" {
		return
	}

	log.Printf("⛏️ Mining %d transport options for route %s -> %s", len(transports), origin, dest)

	for _, t := range transports {
		// Panggil Repo untuk Upsert
		err := s.TransportRepo.UpsertTransportOption(ctx, t, origin, dest)
		if err != nil {
			log.Printf("⚠️ Failed to seed transport: %v", err)
		}
	}
}

func (s *TripService) mineAttractions(ctx context.Context, locID string, itinerary []domain.ItineraryDay) {
	if locID == "" {
		return
	}
	for _, day := range itinerary {
		for _, act := range day.Activities {
			if act.Type != "Logistics" {
				_ = s.AttractionRepo.UpsertAttraction(ctx, domain.TouristAttraction{
					ID: uuid.New().String(), LocationID: locID, Name: act.PlaceName, Category: act.Type, Description: act.Description,
				})
			}
		}
	}
}

func (s *TripService) mineAccommodations(ctx context.Context, locID string, accoms []domain.AccommodationOption) {
	if locID == "" {
		return
	}
	for _, a := range accoms {
		_ = s.AccomRepo.SaveAccommodation(ctx, domain.Accommodation{
			LocationID: locID, Name: a.Name, Type: a.Type, Rating: a.Rating, PricePerNight: a.PricePerNight, ImageURL: a.ImageURL,
		})
	}
}

func (s *TripService) sendEvent(ch chan string, dataType string, data interface{}) {
	payload, _ := json.Marshal(map[string]interface{}{"type": dataType, "data": data})
	ch <- string(payload)
}

// Legacy helpers
func recommendDestination(style string) string {
	switch style {
	case "cultural":
		return "Yogyakarta"
	case "adventure":
		return "Labuan Bajo"
	case "relaxed":
		return "Bali"
	default:
		return "Lombok"
	}
}

func (s *TripService) GetTrip(ctx context.Context, id string) (*domain.TripAndPlan, error) {
	return s.TripRepo.GetTripWithPlan(ctx, id)
}

func (s *TripService) ListTrips(ctx context.Context) ([]domain.Trip, error) {
	return s.TripRepo.GetAllTrips(ctx)
}

func (s *TripService) SubmitFeedback(ctx context.Context, tripID string, req domain.Feedback) error {
	req.TripID = tripID
	req.CreatedAt = time.Now()
	return s.FeedbackRepo.CreateFeedback(ctx, req)
}

func (s *TripService) enrichWithImages(plan *domain.TripPlan) {
	var wg sync.WaitGroup
	for i := range plan.AccommodationOptions {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			query := fmt.Sprintf("%s %s hotel", plan.AccommodationOptions[idx].Name, plan.AccommodationOptions[idx].LocationArea)
			plan.AccommodationOptions[idx].ImageURL = s.ImageSvc.SearchImage(query)
		}(i)
	}
	wg.Wait()
}
