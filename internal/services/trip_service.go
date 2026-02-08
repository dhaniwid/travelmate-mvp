package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type TripService struct {
	TripRepo       *repositories.TripRepository
	FeedbackRepo   *repositories.FeedbackRepository
	AccomRepo      *repositories.AccommodationRepository
	AttractionRepo *repositories.AttractionRepository
	TransportRepo  *repositories.TransportRepository
	PerfRepo       *repositories.PerformanceRepository
	DiscoveryRepo  *repositories.DiscoveryRepository
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
	discoveryRepo *repositories.DiscoveryRepository,
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
		DiscoveryRepo:  discoveryRepo,
		Planner:        p,
		TransportServ:  transportS,
		LocationServ:   locS,
		ImageSvc:       imageSvc,
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

// SaveUserTrip SaveUserTrip: Menyimpan trip final yang dikirim oleh user (setelah review di frontend)
func (s *TripService) SaveUserTrip(ctx context.Context, trip *domain.Trip) error {
	// Validasi Basic
	if trip.ID == "" {
		return fmt.Errorf("trip id is mandatory for saving")
	}
	if trip.UserID == "" {
		return fmt.Errorf("user id is mandatory for saving")
	}

	// [OPSIONAL] Logic Premium Check
	// Di sini kamu bisa cek apakah user Free sudah mencapai limit save trip
	// count, _ := s.TripRepo.CountTripsByUser(trip.UserID)
	// if userIsFree && count >= 3 { return error }

	// Panggil Repo untuk UPDATE (Claim), bukan Create
	return s.TripRepo.ClaimTrip(ctx, trip.ID, trip.UserID, trip.PlanData)
}

func (s *TripService) DeleteUserTrip(ctx context.Context, tripID string, userID string) error {
	return s.TripRepo.Delete(ctx, tripID, userID)
}

func (s *TripService) GetUserTrips(ctx context.Context, userID string) ([]domain.Trip, error) {
	return s.TripRepo.ListTripsByUser(ctx, userID)
}

func (s *TripService) GetDestinationDiscovery(ctx context.Context, city string) (*domain.DiscoveryResponse, error) {
	// 1. Sanitasi Input
	cleanCity := strings.TrimSpace(city)
	if cleanCity == "" {
		return nil, fmt.Errorf("city name cannot be empty")
	}

	// =========================================================
	// 🛑 STEP 1: CEK DATABASE (Mining Result)
	// =========================================================
	// Kita cek apakah kita sudah pernah "menambang" kota ini sebelumnya?
	storedDest, err := s.DiscoveryRepo.GetDestinationByCity(ctx, cleanCity)

	// Syarat Cache Hit: Tidak Error, Data Ada, dan Kolom JSON DiscoveryData tidak kosong
	if err == nil && storedDest != nil && len(storedDest.DiscoveryData) > 0 {
		// fmt.Printf("💎 Cache Hit (DB): Mengambil data '%s' dari Database\n", cleanCity)

		var resp domain.DiscoveryResponse
		// Unmarshal JSONB dari database kembali ke Struct
		if err := json.Unmarshal(storedDest.DiscoveryData, &resp); err == nil {
			return &resp, nil
		}
		// Jika unmarshal gagal, kita anggap cache rusak dan lanjut panggil AI
		log.Printf("⚠️ Corrupt JSON in DB for %s, refreshing...", cleanCity)
	}

	// =========================================================
	// 🤖 STEP 2: AI LAYER (Mining Process)
	// =========================================================
	log.Printf("⛏️ Mining New Data: Asking OpenAI for '%s'...", cleanCity)

	// Panggil AI Planner
	resp, err := s.Planner.GetDiscoveryInfo(ctx, cleanCity)
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery info: %w", err)
	}

	// =========================================================
	// 💾 STEP 3: SIMPAN KE DATABASE (Future Asset)
	// =========================================================
	go func(data *domain.DiscoveryResponse) {
		// Buat context background baru karena context request utama mungkin sudah cancel/selesai
		bgCtx := context.Background()

		// Convert struct response ke JSON Bytes
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			log.Printf("❌ Failed to marshal discovery response for DB: %v", err)
			return
		}

		// Siapkan Metadata untuk disimpan
		meta := domain.DestinationMetadata{
			CityName:    data.City,
			Description: data.Tagline,
			// Simpan JSON mentah AI ke kolom 'discovery_data' (JSONB)
			DiscoveryData: jsonBytes,
		}

		// Panggil Repo Save
		if err := s.DiscoveryRepo.SaveDestination(bgCtx, meta); err != nil {
			log.Printf("⚠️ Failed to save mined data to DB: %v", err)
		} else {
			log.Printf("✅ Successfully mined and saved: %s", data.City)
		}
	}(resp)

	return resp, nil
}
