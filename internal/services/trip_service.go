package services

import (
	"context"
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
	// 1. Pastikan ID ada (jika frontend lupa kirim)
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// 2. Set waktu
	if trip.CreatedAt.IsZero() {
		trip.CreatedAt = time.Now()
	}

	// 3. Panggil Repository
	return s.TripRepo.Create(ctx, trip)
}

func (s *TripService) DeleteUserTrip(ctx context.Context, tripID string, userID string) error {
	return s.TripRepo.Delete(ctx, tripID, userID)
}

func (s *TripService) GetUserTrips(ctx context.Context, userID string) ([]domain.Trip, error) {
	return s.TripRepo.ListTripsByUser(ctx, userID)
}
