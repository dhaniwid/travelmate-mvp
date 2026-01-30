package services

import (
	"strconv"
	"strings"
	"travelmate/internal/repositories"
)

type TransportService struct {
	Repo *repositories.TransportRepository
}

func NewTransportService(repo *repositories.TransportRepository) *TransportService {
	return &TransportService{
		Repo: repo,
	}
}

//func (s *TransportService) SearchRealtimeTickets(ctx context.Context, origin, destination string) ([]domain.TransportOption, error) {
//	log.Printf("🔍 [TransportService] Searching tickets from %s to %s...", origin, destination)
//
//	// 1. Coba Cari di Database dulu (Cache)
//	cachedRoutes, err := s.Repo.GetRoute(ctx, origin, destination)
//	if err == nil && len(cachedRoutes) > 0 {
//		log.Printf("✅ [TransportService] Found %d routes in cache/database!", len(cachedRoutes))
//		var options []domain.TransportOption
//		for _, r := range cachedRoutes {
//			// Convert Domain Route ke TransportOption (utk return ke user)
//			options = append(options, domain.TransportOption{
//				Type:          r.TransportMode,
//				Name:          r.ProviderName,
//				Price:         r.Price,
//				EstimatedTime: formatDuration(r.AvgDurationMins),
//				Pros:          "Best price from historical data",
//			})
//		}
//		return options, nil
//	}
//
//	// 2. Jika tidak ada di DB, Panggil Mock API (Dummy)
//	log.Println("🌐 [TransportService] No cache found. Calling External API (Mock)...")
//	tickets := s.mockExternalAPI(origin, destination)
//
//	// 3. Learning (Simpan hasil Mock ke DB untuk masa depan)
//	go func() {
//		// Pakai background context agar tidak memblokir user
//		bgCtx := context.Background()
//		for _, t := range tickets {
//			// Parse duration string "1h 30m" -> int minutes (simple logic)
//			mins := parseDurationToMins(t.EstimatedTime)
//
//			newRoute := domain.Route{
//				OriginCode:      origin,
//				DestinationCode: destination,
//				TransportMode:   t.Type,
//				ProviderName:    t.Name,
//				Price:           t.Price,
//				AvgDurationMins: mins,
//			}
//
//			_ = s.Repo.SaveRoute(bgCtx, newRoute)
//		}
//	}()
//
//	return tickets, nil
//}

// --- Helper Functions ---

//func (s *TransportService) mockExternalAPI(origin, destination string) []domain.TransportOption {
//	// Simulasi delay API
//	time.Sleep(500 * time.Millisecond)
//
//	basePrice := 1200000 // Default 1.2jt
//
//	// Random variation
//	rand.Seed(time.Now().UnixNano())
//	variance := rand.Intn(400000) - 200000 // +/- 200rb
//
//	priceFlight := int64(basePrice + variance)
//	priceTrain := int64(basePrice/2 + variance)
//
//	return []domain.TransportOption{
//		{
//			Type:          "Flight",
//			Name:          "Garuda Indonesia",
//			Price:         priceFlight,
//			EstimatedTime: "1h 30m",
//			Pros:          "Fastest & Direct",
//		},
//		{
//			Type:          "Train",
//			Name:          "KAI Executive",
//			Price:         priceTrain,
//			EstimatedTime: "6h 00m",
//			Pros:          "Scenic Route",
//		},
//	}
//}

// Helper: Convert "1h 30m" -> 90 (int)
func parseDurationToMins(durationStr string) int {
	durationStr = strings.ToLower(durationStr)
	hours := 0

	if strings.Contains(durationStr, "h") {
		parts := strings.Split(durationStr, "h")
		h, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		hours = h
	}

	return hours * 60 // Return menit
}

func formatDuration(mins int) string {
	hours := mins / 60
	return strconv.Itoa(hours) + "h"
}
