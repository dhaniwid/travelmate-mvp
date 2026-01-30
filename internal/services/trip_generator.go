package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
	"travelmate/internal/domain"
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

	// 6. Finalize
	log.Printf("🏁 [TOTAL TIME] UI Ready in: %v", time.Since(startTime))
	go s.FinalizeAndSaveToDB(trip, finalPlan)
	doneChan <- true
}

// Helper 1: Menangani Surprise Me & Metadata Awal
func (s *TripService) resolveDestination(ctx context.Context, trip domain.Trip, eventChan chan string) (domain.Trip, error) {
	meta := map[string]string{"trip_id": trip.ID}

	if trip.Destination == "" {
		trip.Destination = s.recommendDestination(trip.Style)
		log.Printf("🎲 [SURPRISE ME] Selected: %s", trip.Destination)
		meta["destination"] = trip.Destination
	}

	s.sendEvent(eventChan, "metadata", meta)
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
		itiRes     []domain.ItineraryDay
		logRes     domain.TripPlan
		packingRes []domain.PackingItem
	)

	// --- TASK 1: ITINERARY ---
	s.runAsyncTask(&wg, "TASK 1", eventChan, "itinerary",
		func() (interface{}, error) {
			return s.Planner.GenerateOnlyItinerary(ctx, trip)
		},
		func(res interface{}) {
			itiRes = res.([]domain.ItineraryDay) // Type Assertion
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
	//s.runAsyncTask(&wg, "TASK 3", eventChan, "packing_list",
	//	func() (interface{}, error) {
	//		return s.Planner.GeneratePackingList(ctx, trip)
	//	},
	//	func(res interface{}) {
	//		packingRes = res.([]domain.PackingItem)
	//	},
	//)

	// Tunggu semua selesai
	wg.Wait()

	// Merge Results
	finalPlan := logRes
	finalPlan.TripID = trip.ID
	finalPlan.Itinerary = itiRes
	finalPlan.PackingList = packingRes

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
		start := time.Now()
		log.Printf("🚀 [%s] Starting...", taskName)

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

		log.Printf("✅ [%s] Finished in: %v", taskName, time.Since(start))
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
func (s *TripService) GetPackingList(ctx context.Context, tripID string) ([]domain.PackingItem, error) {
	// 1. Ambil Data Trip (Kita butuh Destinasi, Durasi, dan Style)
	trip, err := s.TripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("trip not found: %w", err)
	}

	// 2. Panggil AI Planner (atau TemplatePlanner jika offline)
	return s.Planner.GeneratePackingList(ctx, *trip)
}

func (s *TripService) sendEvent(ch chan string, dataType string, data interface{}) {
	payload, _ := json.Marshal(map[string]interface{}{"type": dataType, "data": data})
	ch <- string(payload)
}
