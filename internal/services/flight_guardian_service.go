package services

import (
	"context"
	"fmt"
	"time"

	"travelmate/internal/domain"
)

// FlightAlertRepo interface for repository operations
type FlightAlertRepo interface {
	CreateAlert(ctx context.Context, alert *domain.FlightPriceAlert) error
	GetAlertByTripAndOrigin(ctx context.Context, tripID, originAirport string) (*domain.FlightPriceAlert, error)
	GetAlertsByTrip(ctx context.Context, tripID string) ([]*domain.FlightPriceAlert, error)
	UpdatePrice(ctx context.Context, alertID string, newPrice float64) error
	DeactivateAlert(ctx context.Context, alertID string) error
	GetActiveAlerts(ctx context.Context) ([]*domain.FlightPriceAlert, error)
}

// TripRepo interface for trip operations
type TripRepo interface {
	GetByID(ctx context.Context, tripID string) (*domain.Trip, error)
}

// FlightGuardianService handles flight price tracking logic
type FlightGuardianService struct {
	alertRepo FlightAlertRepo
	tripRepo  TripRepo
	amadeus   *AmadeusService
}

// NewFlightGuardianService creates a new guardian service
func NewFlightGuardianService(alertRepo FlightAlertRepo, tripRepo TripRepo, amadeus *AmadeusService) *FlightGuardianService {
	return &FlightGuardianService{
		alertRepo: alertRepo,
		tripRepo:  tripRepo,
		amadeus:   amadeus,
	}
}

// ActivateGuardian activates price monitoring for a trip
func (s *FlightGuardianService) ActivateGuardian(ctx context.Context, tripID, originAirport, destinationAirport string) (*domain.FlightGuardianStatus, error) {
	// 1. Check if alert already exists
	existingAlert, err := s.alertRepo.GetAlertByTripAndOrigin(ctx, tripID, originAirport)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing alert: %w", err)
	}
	if existingAlert != nil {
		// Already tracking
		return &domain.FlightGuardianStatus{
			Status:       "active",
			CurrentPrice: existingAlert.CurrentPrice,
			Currency:     existingAlert.Currency,
			AlertID:      existingAlert.ID,
		}, nil
	}

	// 2. Fetch trip details
	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trip: %w", err)
	}
	if trip == nil {
		return nil, fmt.Errorf("trip not found: %s", tripID)
	}

	// 3. Validate and parse dates
	startDate, err := time.Parse("2006-01-02", trip.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}

	if startDate.Before(time.Now().Add(-24 * time.Hour)) {
		return nil, fmt.Errorf("cannot track flights for past trips")
	}

	// 4. Fetch current price from Amadeus
	departureDate := trip.StartDate // Already in YYYY-MM-DD format
	var returnDatePtr *string
	if trip.TripDays > 0 {
		endDate := startDate.AddDate(0, 0, trip.TripDays)
		returnDate := endDate.Format("2006-01-02")
		returnDatePtr = &returnDate
	}

	price, currency, err := s.amadeus.GetCheapestPrice(ctx, originAirport, destinationAirport, departureDate, returnDatePtr)
	if err != nil {
		return &domain.FlightGuardianStatus{
			Status: "error",
		}, fmt.Errorf("failed to fetch initial price: %w", err)
	}

	// 5. Create alert in database
	now := time.Now()
	var endDatePtr *time.Time
	if trip.TripDays > 0 {
		endDate := startDate.AddDate(0, 0, trip.TripDays)
		endDatePtr = &endDate
	}

	alert := &domain.FlightPriceAlert{
		TripID:             tripID,
		OriginAirport:      originAirport,
		DestinationAirport: destinationAirport,
		DepartureDate:      startDate,
		ReturnDate:         endDatePtr,
		InitialPrice:       price,
		CurrentPrice:       price,
		LowestPriceSeen:    price,
		Currency:           currency,
		LastCheckedAt:      &now,
		IsActive:           true,
	}

	if err := s.alertRepo.CreateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// 6. Return status
	return &domain.FlightGuardianStatus{
		Status:       "active",
		CurrentPrice: price,
		Currency:     currency,
		AlertID:      alert.ID,
	}, nil
}

// GetTripAlerts retrieves all alerts for a trip
func (s *FlightGuardianService) GetTripAlerts(ctx context.Context, tripID string) ([]*domain.FlightPriceAlert, error) {
	return s.alertRepo.GetAlertsByTrip(ctx, tripID)
}

// DeactivateGuardian deactivates price monitoring
func (s *FlightGuardianService) DeactivateGuardian(ctx context.Context, alertID string) error {
	return s.alertRepo.DeactivateAlert(ctx, alertID)
}

// CheckAndUpdatePrices checks all active alerts and updates prices (for background worker)
func (s *FlightGuardianService) CheckAndUpdatePrices(ctx context.Context) error {
	alerts, err := s.alertRepo.GetActiveAlerts(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active alerts: %w", err)
	}

	for _, alert := range alerts {
		// Fetch current price
		departureDate := alert.DepartureDate.Format("2006-01-02")
		var returnDatePtr *string
		if alert.ReturnDate != nil {
			returnDate := alert.ReturnDate.Format("2006-01-02")
			returnDatePtr = &returnDate
		}

		price, _, err := s.amadeus.GetCheapestPrice(ctx, alert.OriginAirport, alert.DestinationAirport, departureDate, returnDatePtr)
		if err != nil {
			// Log error but continue with other alerts
			fmt.Printf("Failed to check price for alert %s: %v\n", alert.ID, err)
			continue
		}

		// Update database
		if err := s.alertRepo.UpdatePrice(ctx, alert.ID, price); err != nil {
			fmt.Printf("Failed to update price for alert %s: %v\n", alert.ID, err)
			continue
		}

		// TODO: Check if we need to send notification (price drop > threshold)
	}

	return nil
}

// SearchLocations wraps Amadeus search
func (s *FlightGuardianService) SearchLocations(ctx context.Context, keyword string) ([]Location, error) {
	return s.amadeus.SearchLocations(ctx, keyword)
}

// SearchFlightOffers wraps Amadeus search
func (s *FlightGuardianService) SearchFlightOffers(ctx context.Context, origin, dest, date string, returnDate *string) ([]FlightOfferDetail, error) {
	return s.amadeus.SearchFlightOffersDetail(ctx, origin, dest, date, returnDate)
}
