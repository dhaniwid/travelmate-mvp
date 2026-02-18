package scheduler

import (
	"context"
	"log"
	"time"

	"travelmate/internal/services"

	"github.com/robfig/cron/v3"
)

// FlightGuardianScheduler manages periodic price checks
type FlightGuardianScheduler struct {
	cron            *cron.Cron
	guardianService *services.FlightGuardianService
}

// NewFlightGuardianScheduler creates a new scheduler
func NewFlightGuardianScheduler(guardianService *services.FlightGuardianService) *FlightGuardianScheduler {
	return &FlightGuardianScheduler{
		cron:            cron.New(),
		guardianService: guardianService,
	}
}

// Start begins the scheduled price checking
func (s *FlightGuardianScheduler) Start() error {
	// Schedule: Every 12 hours
	_, err := s.cron.AddFunc("0 */12 * * *", func() {
		s.checkPrices()
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("🦅 Flight Guardian Scheduler: Started (checking every 12 hours)")
	return nil
}

// Stop gracefully stops the scheduler
func (s *FlightGuardianScheduler) Stop() {
	s.cron.Stop()
	log.Println("🦅 Flight Guardian Scheduler: Stopped")
}

// checkPrices runs the background price check
func (s *FlightGuardianScheduler) checkPrices() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Println("🦅 Flight Guardian: Starting price check...")
	startTime := time.Now()

	err := s.guardianService.CheckAndUpdatePrices(ctx)
	if err != nil {
		log.Printf("🦅 Flight Guardian: Error during price check: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("🦅 Flight Guardian: Price check completed in %v", duration)
}

// RunNow triggers an immediate check (for testing or manual runs)
func (s *FlightGuardianScheduler) RunNow() {
	go s.checkPrices()
}
