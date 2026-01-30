package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"sync"
	"travelmate/internal/domain"
)

// --- HELPERS & MINING FUNCTIONS ---

func (s *TripService) FinalizeAndSaveToDB(trip domain.Trip, plan domain.TripPlan) {
	ctx := context.Background()

	// 1. Simpan Trip Plan user (Prioritas Utama)
	if err := s.TripRepo.SaveTripPlan(ctx, trip, plan); err != nil {
		log.Printf("❌ Failed to save user trip plan: %v", err)
		// Jangan return, coba lanjut mining sebisa mungkin
	}

	// 🛡️ SECURITY CHECK: Pastikan LocationID valid sebelum mining
	// Jika kosong, coba cari lagi berdasarkan nama destination
	if trip.LocationID == "" {
		log.Println("⚠️ LocationID is missing for mining. Attempting resolve...")
		loc, _ := s.LocationServ.GetOrEnrichLocation(ctx, trip.Destination)
		if loc != nil {
			trip.LocationID = loc.ID
		} else {
			log.Println("❌ Mining aborted: LocationID not found.")
			return
		}
	}

	// 2. Mining Accommodations (Seed DB)
	//go s.mineAccommodations(ctx, trip.LocationID, plan.AccommodationOptions)

	// 3. Mining Attractions (Seed DB)
	go s.mineAttractions(ctx, trip.LocationID, plan.Itinerary)

	// 4. Mining Transports (Seed DB)
	go s.mineTransports(ctx, trip.Origin, trip.Destination, plan.TransportOptions)
}

func (s *TripService) mineTransports(ctx context.Context, origin, dest string, transports []domain.TransportOption) {
	if origin == "" || dest == "" {
		return
	}

	log.Printf("⛏️ Mining %d transport options for route %s -> %s", len(transports), origin, dest)

	//for _, t := range transports {
	//	// Panggil Repo untuk Upsert
	//	err := s.TransportRepo.UpsertTransportOption(ctx, t, origin, dest)
	//	if err != nil {
	//		log.Printf("⚠️ Failed to seed transport: %v", err)
	//	}
	//}
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

//func (s *TripService) mineAccommodations(ctx context.Context, locID string, accoms []domain.AccommodationOption) {
//	if locID == "" {
//		return
//	}
//	for _, a := range accoms {
//		_ = s.AccomRepo.SaveAccommodation(ctx, domain.Accommodation{
//			LocationID: locID, Name: a.Name, Type: a.Type, Rating: a.Rating, PricePerNight: a.PricePerNight, ImageURL: a.ImageURL,
//		})
//	}
//}

// CalculateBudget menghitung total biaya berdasarkan opsi logistik
//func (s *TripService) CalculateBudget(tripDays int, transport []domain.TransportOption, hotels []domain.AccommodationOption) domain.BudgetBreakdown {
//	var totalTransport int64 = 0
//	var totalAccom int64 = 0
//
//	// 1. Hitung Rata-rata Transport
//	if len(transport) > 0 {
//		var sum int64
//		for _, t := range transport {
//			sum += int64(t.Price)
//		}
//		totalTransport = sum / int64(len(transport))
//	}
//
//	// 2. Hitung Total Akomodasi (Harga per malam * Durasi)
//	if len(hotels) > 0 {
//		var sum int64
//		for _, h := range hotels {
//			sum += int64(h.PricePerNight)
//		}
//		avgNight := sum / int64(len(hotels))
//		totalAccom = avgNight * int64(tripDays)
//	}
//
//	// 3. Estimasi Biaya Harian (Hardcoded Baseline)
//	const (
//		DailyFoodCost   = 150000 // Makan 3x50rb
//		DailyTicketCost = 100000 // Tiket wisata rata-rata
//		DailyMiscCost   = 50000  // Jaga-jaga
//	)
//
//	return domain.BudgetBreakdown{
//		Transport:     domain.FlexibleInt64(totalTransport),
//		Accommodation: domain.FlexibleInt64(totalAccom),
//		Food:          domain.FlexibleInt64(int64(DailyFoodCost * tripDays)),
//		Tickets:       domain.FlexibleInt64(int64(DailyTicketCost * tripDays)),
//		Misc:          domain.FlexibleInt64(int64(DailyMiscCost * tripDays)),
//	}
//}

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
