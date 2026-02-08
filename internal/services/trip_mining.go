package services

import (
	"context"
	"log"
	"travelmate/internal/domain"

	"github.com/google/uuid"
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

func (s *TripService) mineTransports(_ context.Context, origin, dest string, transports []domain.TransportOption) {
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
