package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
	"travelmate/internal/domain"

	"github.com/google/uuid"
)

// GenerateTripStream ==========================================================
// 1. STREAM FUNCTION (Parallel & Responsive)
// ==========================================================
// GenerateTripStream bertindak sebagai "Conductor" / Orchestrator saja.
func (s *TripService) GenerateTripStream(ctx context.Context, trip domain.Trip, eventChan chan string, doneChan chan bool) {
	startTime := time.Now()

	// 1. Setup Trip ID
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// 2. Resolve Destination (Surprise Me logic)
	trip, err := s.resolveDestination(ctx, trip, eventChan)
	if err != nil {
		log.Printf("Error resolving destination: %v", err)
		return
	}

	// 3. Check Cache (Fast Path)
	cached, err := s.checkCache(ctx, trip, eventChan)
	if cached {
		doneChan <- true
		log.Printf("🏁 [FAST PATH] UI Ready in: %v", time.Since(startTime))
		return
	}

	// 4. Enrich Location Data (Lat/Long)
	trip = s.enrichLocation(ctx, trip)

	// 5. Execute AI Tasks in Parallel (Heavy Lifting)
	finalPlan := s.executeAIPlannerParallel(ctx, trip, eventChan)

	// 6. Save to Database (MUST be synchronous before redirect!)
	s.FinalizeAndSaveToDB(trip, finalPlan)

	// 6.5. Send itinerary_ready AFTER DB save to prevent 404 race condition
	s.sendEvent(eventChan, "itinerary_ready", map[string]interface{}{
		"trip_id": trip.ID,
		"message": "Trip saved to database. Safe to redirect.",
	})

	// 7. Background: Enrich Activities (Photos/PlaceID)
	go func(tid string) {
		enrichCtx := context.Background() // New context for bg job
		s.EnrichmentSvc.EnrichTrip(enrichCtx, tid)
	}(trip.ID)

	// 8. Finalize
	log.Printf("🏁 [TOTAL TIME] UI Ready in: %v", time.Since(startTime))
	doneChan <- true
}

// Helper 1: Menangani Surprise Me & Metadata Awal
func (s *TripService) resolveDestination(_ context.Context, trip domain.Trip, eventChan chan string) (domain.Trip, error) {
	if trip.Destination == "" {
		trip.Destination = s.recommendDestination(trip.Style)
		log.Printf("🎲 [SURPRISE ME] Selected: %s", trip.Destination)
	}

	s.sendEvent(eventChan, "metadata", trip)
	return trip, nil
}

// Helper 2: Cek Cache Database
func (s *TripService) checkCache(ctx context.Context, trip domain.Trip, eventChan chan string) (bool, error) {
	existingPlan, err := s.TripRepo.GetExistingPlanByCriteria(ctx, trip.Destination, trip.Style, trip.TripDays)

	if err == nil && existingPlan != nil {
		log.Printf("⚡ [FAST PATH] Found cache for %s", trip.Destination)
		s.sendEvent(eventChan, "itinerary", existingPlan.Itinerary)
		s.sendEvent(eventChan, "logistics", existingPlan)
		s.sendEvent(eventChan, "packing_list", existingPlan.PackingList)
		return true, nil
	}
	return false, nil
}

// Helper 3: Update Data Lokasi
func (s *TripService) enrichLocation(ctx context.Context, trip domain.Trip) domain.Trip {
	locData, _ := s.LocationServ.GetOrEnrichLocation(ctx, trip.Destination)
	if locData != nil {
		trip.LocationID = locData.ID
		trip.Destination = locData.Name
	}
	return trip
}

// Helper 4: Core AI Parallel Execution
func (s *TripService) executeAIPlannerParallel(ctx context.Context, trip domain.Trip, eventChan chan string) domain.TripPlan {
	var wg sync.WaitGroup

	// Wadah hasil
	var (
		itiRes     domain.ItineraryResponse
		logRes     domain.TripPlan
		packingRes []domain.PackingCategory
	)

	// --- TASK 1: ITINERARY ---
	s.runAsyncTask(&wg, "TASK 1", eventChan, "itinerary",
		func() (interface{}, error) {
			return s.Planner.GenerateOnlyItinerary(ctx, trip)
		},
		func(res interface{}) {
			itiRes = res.(domain.ItineraryResponse) // Type Assertion
			// Note: itinerary_ready event moved to AFTER database save to prevent 404 race condition
		},
	)

	// --- TASK 2: LOGISTICS ---
	s.runAsyncTask(&wg, "TASK 2", eventChan, "logistics",
		func() (interface{}, error) {
			// Logic kalkulasi budget tetap terisolasi di sini
			plan, err := s.Planner.GenerateTransportAndStay(ctx, trip)
			if err != nil {
				return nil, err
			}

			if plan.TransportOptions == nil {
				plan.TransportOptions = []domain.TransportOption{}
			}
			if plan.AccommodationOptions == nil {
				plan.AccommodationOptions = []domain.AccommodationOption{}
			}

			//plan.BudgetBreakdown = s.CalculateBudget(trip.TripDays, plan.TransportOptions, plan.AccommodationOptions)
			return plan, nil
		},
		func(res interface{}) {
			logRes = res.(domain.TripPlan)
		},
	)

	// --- TASK 3: PACKING LIST ---
	s.runAsyncTask(&wg, "TASK 3", eventChan, "packing_list",
		func() (interface{}, error) {
			return s.Planner.GeneratePackingList(ctx, trip)
		},
		func(res interface{}) {
			if val, ok := res.([]domain.PackingCategory); ok {
				packingRes = val
			}
		},
	)

	// --- TASK 4: EDITORIAL (New Phase 2) ---
	var editorialRes domain.EditorialResponse
	s.runAsyncTask(&wg, "TASK 4", eventChan, "editorial",
		func() (interface{}, error) {
			return s.Planner.GenerateEditorial(ctx, trip)
		},
		func(res interface{}) {
			editorialRes = res.(domain.EditorialResponse)
		},
	)

	// Tunggu semua selesai
	wg.Wait()

	// Merge Results
	finalPlan := logRes
	finalPlan.TripID = trip.ID

	// Merge Itinerary (Task 1)
	finalPlan.Itinerary = itiRes.Itinerary

	// Merge Packing (Task 3)
	finalPlan.PackingList = packingRes

	// Merge Editorial (Task 4)
	finalPlan.MorningBriefing = editorialRes.MorningBriefing
	finalPlan.Highlights = editorialRes.Highlights
	finalPlan.Tagline = editorialRes.Tagline
	finalPlan.Vibes = editorialRes.Vibes
	finalPlan.CulinarySignature = editorialRes.CulinarySignature
	finalPlan.HiddenGem = editorialRes.HiddenGem
	finalPlan.HistorySnippet = editorialRes.HistorySnippet

	return finalPlan
}

// Helper generic untuk menjalankan task AI secara paralel
func (s *TripService) runAsyncTask(
	wg *sync.WaitGroup,
	taskName string,
	eventChan chan string,
	eventKey string,
	action func() (interface{}, error), // Fungsi eksekusi utama
	onSuccess func(interface{}), // Callback saat sukses (untuk assign variable)
) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// 1. Eksekusi Action
		result, err := action()
		if err != nil {
			log.Printf("❌ [%s] Error: %v", taskName, err)
			return
		}

		// 2. Kirim Event ke Frontend
		s.sendEvent(eventChan, eventKey, result)

		// 3. Update Variable di Parent (via Callback)
		if onSuccess != nil {
			onSuccess(result)
		}

		if onSuccess != nil {
			onSuccess(result)
		}
	}()
}

// recommendDestination: Simple logic untuk "Surprise Me"
// Jika nanti sudah advance, ini bisa diganti dengan call ke AI kecil (GPT-3.5)
func (s *TripService) recommendDestination(style string) string {
	// Dictionary rekomendasi sederhana
	recommendations := map[string][]string{
		"general":   {"Bali", "Yogyakarta", "Singapore", "Bangkok", "Tokyo"},
		"cultural":  {"Yogyakarta", "Kyoto", "Istanbul", "Rome", "Ubud"},
		"relaxed":   {"Bali", "Maldives", "Phuket", "Lombok", "Sumba"},
		"adventure": {"Labuan Bajo", "Bromo", "Nepal", "New Zealand", "Raja Ampat"},
		"foodie":    {"Osaka", "Penang", "Bandung", "Hanoi", "Padang"},
		"luxury":    {"Paris", "Dubai", "Swiss", "Monaco", "Nusa Dua"},
	}

	// Ambil list sesuai style, default ke 'general' jika style tidak dikenal
	list, exists := recommendations[strings.ToLower(style)]
	if !exists {
		list = recommendations["general"]
	}

	// Pilih random
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(list))

	return list[randomIndex]
}

func (s *TripService) GetActivityAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	return s.Planner.GenerateAlternatives(ctx, dest, activity, location, tags)
}

// GetPackingList mengambil data trip lalu meminta AI membuatkan daftar bawaan
func (s *TripService) GetPackingList(ctx context.Context, tripID string) ([]domain.PackingCategory, error) {
	// 1. Ambil Data Trip (Kita butuh Destinasi, Durasi, dan Style)
	trip, err := s.TripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("trip not found: %w", err)
	}

	// 2. Panggil AI Planner (atau TemplatePlanner jika offline)
	return s.Planner.GeneratePackingList(ctx, *trip)
}

// GenerateTripAsync ==========================================================
// NEW: Progressive Generation (M-123)
// Flow: Fast Init -> Return 200 -> Async Enrichment
// ==============================================================================
func (s *TripService) GenerateTripAsync(ctx context.Context, trip domain.Trip) (*domain.Trip, error) {
	startTime := time.Now()
	// 1. Setup Trip ID
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// 🛡️ SECURITY: REDUNDANT QUOTA CHECK (Final Guard before AI)
	if trip.UserID != "" && trip.UserID != "guest" {
		allowed, err := s.SubService.CheckQuotaAvailability(ctx, trip.UserID)
		if err != nil {
			return nil, fmt.Errorf("quota verification failed: %w", err)
		}
		if !allowed {
			return nil, fmt.Errorf("quota_exceeded: monthly limit reached")
		}
	}

	// 2. Resolve Destination (Surprise Me)
	if strings.EqualFold(trip.Destination, "Surprise") {
		curatedCities := []string{
			"Kyoto, Japan",
			"Bali, Indonesia",
			"Paris, France",
			"Reykjavik, Iceland",
			"Seoul, South Korea",
			"Amsterdam, Netherlands",
			"Sydney, Australia",
			"Cape Town, South Africa",
			"Barcelona, Spain",
			"Singapore",
		}
		rand.Seed(time.Now().UnixNano())
		trip.Destination = curatedCities[rand.Intn(len(curatedCities))]
		log.Printf("🎲 [LOGIC] Surprise Mode activated! Selected: %s", trip.Destination)
	}

	if trip.Destination == "" {
		trip.Destination = s.recommendDestination(trip.Style)
	}

	// Enrich Location (Lat/Long/ID) - Keep sync as it is fast and critical for DB
	trip = s.enrichLocation(ctx, trip)

	// --- NEW: SKELETON-FIRST GENERATION (M-126) ---
	log.Printf("🚀 [SKELETON-FIRST] Starting Parallel Skeleton & Logistics for %s", trip.ID)

	var wg sync.WaitGroup
	var (
		itiRes domain.ItineraryResponse
		itiErr error
		logRes domain.TripLogisticsResponse
		logErr error
	)

	wg.Add(2)
	// Task 1: Skeleton Itinerary (Real Names, Geo-Hints, Hooks)
	go func() {
		defer wg.Done()
		itiRes, itiErr = s.Planner.GenerateTripSkeleton(ctx, trip)
	}()

	// Task 2: Strategic Logistics (Visa, Accommo, Budget)
	go func() {
		defer wg.Done()
		logRes, logErr = s.Planner.GenerateTripLogistics(ctx, trip)
	}()

	wg.Wait()

	if itiErr != nil {
		return nil, fmt.Errorf("skeleton generation failed: %w", itiErr)
	}
	if logErr != nil {
		log.Printf("⚠️ [LOGISTICS] Stage 1 failed: %v", logErr)
	}

	// Map Overview to PlanData
	fullPlan := domain.TripPlan{
		TripID:               trip.ID,
		Itinerary:            itiRes.Itinerary,
		ArrivalGuide:         &logRes.ArrivalGuide,
		BudgetBreakdown:      logRes.BudgetBreakdown,
		PackingList:          logRes.PackingList,
		TransportOptions:     []domain.TransportOption{},
		AccommodationOptions: logRes.StrategicAccommodation,
		Highlights:           []domain.TripHighlight{}, // To be enriched on-demand or background
		MorningBriefing:      "",                       // Lazy load
		Tagline:              trip.Destination + " Escape",
	}

	// 3. Save Trip with Skeleton Plan to DB
	trip.Status = "UPCOMING"
	trip.EnrichmentStatus = domain.EnrichmentStatusPending
	trip.ItineraryStatus = domain.ItineraryStatusCompleted // Skeleton is the primary itinerary now
	trip.PlanData = &fullPlan
	trip.CreatedAt = time.Now()

	if err := s.TripRepo.Create(ctx, &trip); err != nil {
		return nil, fmt.Errorf("failed to save skeleton trip: %w", err)
	}

	log.Printf("📍 [SKELETON-FIRST] Skeleton Plan saved for trip %s. Response time: %v", trip.ID, time.Since(startTime))

	// No background EnrichmentPipeline here! Everything is Lazy Load now.
	return &trip, nil
}

// ProcessProgressiveGeneration handles the multi-phase AI work in background
func (s *TripService) ProcessProgressiveGeneration(tripID string, trip domain.Trip) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("🚀 [PROGRESSIVE] Starting background generation for %s", tripID)

	// --- STAGE 1: Core Itinerary (Sync) ---
	coreRes, err := s.Planner.GenerateTripCore(ctx, trip)
	if err != nil {
		log.Printf("❌ [PROGRESSIVE] Stage 1 Failed: %v", err)
		return
	}

	// Construct skeleton plan
	skeletonPlan := domain.TripPlan{
		TripID:               tripID,
		Itinerary:            coreRes.Itinerary,
		ArrivalGuide:         &domain.ArrivalGuide{},
		BudgetBreakdown:      domain.BudgetBreakdown{},
		PackingList:          []domain.PackingCategory{},
		TransportOptions:     []domain.TransportOption{},
		AccommodationOptions: []domain.AccommodationOption{},
	}

	// Update DB with Stage 1 result
	trip.PlanData = &skeletonPlan
	if err := s.TripRepo.SaveTripPlan(ctx, trip, skeletonPlan); err != nil {
		log.Printf("❌ [PROGRESSIVE] Failed to save Stage 1 results: %v", err)
	}

	// --- STAGES 2 & 3: Deep Enrichment ---
	s.RunEnrichmentPipeline(tripID, coreRes)
}

// RunEnrichmentPipeline (M-125 + M-120): Stages 2 (Vibe) and 3 (Logistics) with Caching
func (s *TripService) RunEnrichmentPipeline(tripID string, stage1Result domain.ItineraryResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("✨ [PIPELINE] Starting Enrichment Pipeline for Trip %s", tripID)
	startTime := time.Now()

	// Fetch trip metadata for context
	tripAndPlan, err := s.TripRepo.GetTripWithPlan(ctx, tripID)
	if err != nil || tripAndPlan == nil {
		log.Printf("❌ [PIPELINE] Trip not found: %v", err)
		return
	}
	trip := tripAndPlan.Trip
	plan := tripAndPlan.Plan

	// --- STEP A: BATCH CHECK CACHE (M-120) ---
	var missingActivities []struct {
		Day           int
		ActivityIndex int
		Activity      domain.Activity
	}
	cachedCount := 0

	for dIdx, day := range plan.Itinerary {
		for aIdx, act := range day.Activities {
			if act.Type == "Logistics" {
				continue
			}

			// Check Cache
			cached, err := s.AttractionRepo.GetByName(ctx, act.PlaceName, trip.LocationID)
			if err == nil && cached != nil {
				// Populate from Cache
				plan.Itinerary[dIdx].Activities[aIdx].Description = cached.Description
				plan.Itinerary[dIdx].Activities[aIdx].Latitude = &cached.Latitude
				plan.Itinerary[dIdx].Activities[aIdx].Longitude = &cached.Longitude
				plan.Itinerary[dIdx].Activities[aIdx].PlaceID = cached.PlaceID
				plan.Itinerary[dIdx].Activities[aIdx].TransitTime = cached.VisitDuration // Reusing visit_duration for now
				if len(cached.Photos) > 0 {
					plan.Itinerary[dIdx].Activities[aIdx].ImageURL = cached.Photos[0]
				}
				cachedCount++
				log.Printf("🎯 [CACHE HIT] Found details for: %s", act.PlaceName)
			} else {
				// Mark for AI Enrichment
				missingActivities = append(missingActivities, struct {
					Day           int
					ActivityIndex int
					Activity      domain.Activity
				}{
					Day:           day.Day,
					ActivityIndex: aIdx,
					Activity:      act,
				})
			}
		}
	}

	log.Printf("📊 [CACHE STATS] Hits: %d | Missing: %d", cachedCount, len(missingActivities))

	// --- STEP B: SMART PROMPTING (Parallel Stage 2 & 3) ---
	var (
		vibeRes domain.TripVibeResponse
		vibeErr error
		logRes  domain.TripLogisticsResponse
		logErr  error
		pewg    sync.WaitGroup
	)

	pewg.Add(2)

	// Stage 2: Enrichment (Vibe/Descriptions)
	go func() {
		defer pewg.Done()
		if len(missingActivities) > 0 {
			// Construct a minimal "Needs Enrichment" JSON
			partialItinerary := domain.ItineraryResponse{
				Itinerary: []domain.ItineraryDay{},
			}
			// Group missing into days for prompt context
			dayMap := make(map[int][]domain.Activity)
			for _, m := range missingActivities {
				dayMap[m.Day] = append(dayMap[m.Day], m.Activity)
			}
			for dayNum, acts := range dayMap {
				partialItinerary.Itinerary = append(partialItinerary.Itinerary, domain.ItineraryDay{
					Day:        dayNum,
					Activities: acts,
				})
			}

			s1Bytes, _ := json.Marshal(partialItinerary)
			log.Printf("🧠 [AI] Calling TRIP_ENRICHMENT for %d new activities", len(missingActivities))
			vibeRes, vibeErr = s.Planner.EnrichTripVibe(ctx, string(s1Bytes))
		} else {
			log.Printf("✅ [AI SKIP] All activities cached. Generating Highlights/Briefings only.")
			s1Bytes, _ := json.Marshal(stage1Result)
			vibeRes, vibeErr = s.Planner.EnrichTripVibe(ctx, string(s1Bytes))
		}
	}()

	// Stage 3: Logistics
	go func() {
		defer pewg.Done()
		logRes, logErr = s.Planner.GenerateTripLogistics(ctx, trip)
	}()

	pewg.Wait()

	if vibeErr != nil {
		log.Printf("⚠️ [PIPELINE] Stage 2 (Vibe) failed: %v", vibeErr)
	}
	if logErr != nil {
		log.Printf("⚠️ [PIPELINE] Stage 3 (Logistics) failed: %v", logErr)
	}

	// --- STEP C: SEED BACK & MERGE ---
	// Merge Stage 2 (Vibe)
	if vibeErr == nil {
		// Update descriptions and SEED BACK to cache
		for _, update := range vibeRes.ItineraryUpdates {
			// Find original activity to map back
			updateDay := update.Day
			updateIdx := update.ActivityIndex

			// If we used partialItinerary, the indices might not match 1:1 with the full plan
			// But the prompt returned day + index relative to partial.
			// To be safe, let's map by Day and Activity Match if index is shaky.
			// For now, assume AI returned Day + Index relative to the INPUT s1Bytes.

			// We need to find which activity in the FULL PLAN this corresponds to.
			// If we sent partial, we need to find the match.

			// Simple fallback: Loop and match by name
			for dIdx, day := range plan.Itinerary {
				if day.Day == updateDay {
					// In Stage 2 prompt, user asked for Index.
					// If we sent grouped days, the index is relative to that day's list in the input.
					if updateIdx >= 0 && updateIdx < len(day.Activities) {
						// Match!
						plan.Itinerary[dIdx].Activities[updateIdx].Description = update.Description

						// --- SEED BACK (M-120) ---
						go func(act domain.Activity) {
							seed := domain.TouristAttraction{
								ID:            uuid.New().String(),
								LocationID:    trip.LocationID,
								Name:          act.PlaceName,
								Description:   act.Description,
								Latitude:      *act.Latitude,
								Longitude:     *act.Longitude,
								VisitDuration: update.VisitDuration, // from AI update
								Category:      update.Category,      // from AI update
							}
							if err := s.AttractionRepo.UpsertAttraction(context.Background(), seed); err != nil {
								log.Printf("⚠️ [SEED] Failed to seed %s: %v", seed.Name, err)
							} else {
								log.Printf("🌱 [SEED] Cached: %s", seed.Name)
							}
						}(plan.Itinerary[dIdx].Activities[updateIdx])
					}
				}
			}
		}

		// Update morning briefings
		for _, briefing := range vibeRes.MorningBriefings {
			if briefing.Day > 0 && briefing.Day <= len(plan.Itinerary) {
				plan.Itinerary[briefing.Day-1].MorningBriefing = &domain.MorningBriefing{
					WeatherForecast: briefing.WeatherForecast,
					OutfitTip:       briefing.OutfitTip,
					LocalVibe:       briefing.LocalVibe,
				}
			}
		}
		plan.Highlights = vibeRes.Highlights
	}

	// Merge Stage 3 (Logistics)
	if logErr == nil {
		plan.ArrivalGuide = &logRes.ArrivalGuide
		plan.BudgetBreakdown = logRes.BudgetBreakdown
		plan.PackingList = logRes.PackingList
		plan.AccommodationOptions = logRes.StrategicAccommodation
	}

	// Save back to DB
	trip.PlanData = &plan
	trip.EnrichmentStatus = domain.EnrichmentStatusCompleted

	if err := s.TripRepo.SaveTripPlan(ctx, trip, plan); err != nil {
		log.Printf("❌ [PIPELINE] Final save failed: %v", err)
		return
	}

	// Post-pipeline: Trigger photo enrichment if needed
	go s.EnrichmentSvc.EnrichTrip(context.Background(), tripID)

	log.Printf("✅ [PIPELINE] Pipeline Completed for %s in %v", tripID, time.Since(startTime))
}

// GenerateDetailedItineraryBackground (Phase 2): Background worker for detailed schedule
func (s *TripService) GenerateDetailedItineraryBackground(tripID string, trip domain.Trip, overviewJSON string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("🚀 [STAGE 2] Starting Background Detailed Itinerary for %s", tripID)

	// UPDATE STATUS to 'generating'
	trip.ItineraryStatus = domain.ItineraryStatusGenerating
	_ = s.TripRepo.SaveTripPlan(ctx, trip, *trip.PlanData) // Minimal update

	// 1. Generate Itinerary
	itiRes, err := s.Planner.GenerateTripItinerary(ctx, trip, overviewJSON)
	if err != nil {
		log.Printf("❌ [STAGE 2] Failed: %v", err)
		return
	}

	// 2. Fetch existing plan to merge
	tripAndPlan, err := s.TripRepo.GetTripWithPlan(ctx, tripID)
	if err != nil || tripAndPlan == nil {
		log.Printf("❌ [STAGE 2] Trip not found during merge: %v", err)
		return
	}
	finalPlan := tripAndPlan.Plan
	finalTrip := tripAndPlan.Trip

	// 3. Merge Itinerary
	finalPlan.Itinerary = itiRes.Itinerary
	finalTrip.ItineraryStatus = domain.ItineraryStatusCompleted

	// 4. Save to DB
	if err := s.TripRepo.SaveTripPlan(ctx, finalTrip, finalPlan); err != nil {
		log.Printf("❌ [STAGE 2] Final save failed: %v", err)
		return
	}

	log.Printf("✅ [STAGE 2] Detailed Itinerary Completed for %s", tripID)

	// 5. Trigger Enrichment Pipeline (Stage 3 & Cache Seeding)
	// Reusing EnrichmentPipeline for descriptions/photos if needed
	go s.RunEnrichmentPipeline(tripID, itiRes)
}

func (s *TripService) sendEvent(ch chan string, dataType string, data interface{}) {
	payload, _ := json.Marshal(map[string]interface{}{"type": dataType, "data": data})
	ch <- string(payload)
}
