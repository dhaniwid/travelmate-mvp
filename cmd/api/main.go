package main

import (
	"log"
	"travelmate/internal/config"
	"travelmate/internal/db"
	"travelmate/internal/http"
	"travelmate/internal/http/handlers"
	"travelmate/internal/repositories"
	"travelmate/internal/scheduler"
	"travelmate/internal/services"
	stripePkg "travelmate/internal/stripe"

	_ "github.com/lib/pq"
)

func main() {
	// 1. Config
	cfg := config.LoadConfig()

	// 2. Database
	database := db.Connect(cfg.DBUrl)
	defer database.Close()

	// 3. Repositories
	tripRepo := repositories.NewTripRepository(database)
	fbRepo := repositories.NewFeedbackRepository(database)
	locRepo := repositories.NewLocationRepository(database)
	transportRepo := repositories.NewTransportRepository(database)
	accommodationRepo := repositories.NewAccommodationRepository(database)
	attractionRepo := repositories.NewAttractionRepository(database)
	perfRepo := repositories.NewPerformanceRepository(database)
	perfRepo.PrintStartupDashboard()
	discoveryRepo := repositories.NewDiscoveryRepo(database)
	destRepo := repositories.NewDestinationRepository(database)
	userRepo := repositories.NewUserRepository(database)
	subRepo := repositories.NewSubscriptionRepository(database)
	prefRepo := repositories.NewPreferencesRepository(database)
	placeLibRepo := repositories.NewPlaceLibraryRepository(database)
	analyticsRepo := repositories.NewAnalyticsRepository(database)     // NEW
	collabRepo := repositories.NewCollaboratorRepository(database)     // Collaboration 🤝
	referralRepo := repositories.NewReferralRepository(database)       // Referral System 🎁
	flightAlertRepo := repositories.NewFlightAlertRepository(database) // Flight Guardian ✈️

	// 4. Services (Dependency Injection)
	promptService := services.NewPromptService(database)
	imageSvc := services.NewImageService(cfg.GoogleAPIKey, cfg.GoogleCXId)
	pdfSvc := services.NewPDFService() // NEW

	// Stripe Client
	stripeClient := stripePkg.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookKey)

	subService := services.NewSubscriptionService(userRepo, subRepo, stripeClient)
	discoveryService := services.NewDiscoveryService(destRepo)
	analyticsService := services.NewAnalyticsService(analyticsRepo)                   // NEW
	collabService := services.NewCollaborationService(collabRepo, userRepo, tripRepo) // Collaboration 🤝
	referralService := services.NewReferralService(referralRepo, userRepo)            // Referral System 🎁

	// Flight Guardian ✈️
	amadeusService := services.NewAmadeusService()
	flightGuardianService := services.NewFlightGuardianService(flightAlertRepo, tripRepo, amadeusService)

	var plannerEngine services.PlannerEngine
	if cfg.OpenAIKey != "" {
		plannerEngine = services.NewAIPlanner(cfg.OpenAIKey, promptService, prefRepo, amadeusService) // Pass amadeusService
	} else {
		plannerEngine = services.NewTemplatePlanner()
	}

	// 4.3 Location Service needs promptService
	locationService := services.NewLocationService(locRepo, promptService, cfg.OpenAIKey, imageSvc)

	// Enrichment Service
	enrichService := services.NewEnrichmentService(tripRepo, placeLibRepo, cfg.GoogleAPIKey)

	transportService := services.NewTransportService(transportRepo)

	tripService := services.NewTripService(tripRepo, fbRepo, accommodationRepo, attractionRepo, transportRepo,
		perfRepo, discoveryRepo, plannerEngine, locationService, transportService, imageSvc, pdfSvc, enrichService, subService)

	// 5. Handlers
	tripHandler := handlers.NewTripHandler(tripService, subService, collabRepo)
	fbHandler := handlers.NewFeedbackHandler(tripService)
	subHandler := handlers.NewSubscriptionHandler(subService)
	webhookHandler := handlers.NewWebhookHandler(subService, stripeClient)
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)
	prefHandler := handlers.NewPreferencesHandler(prefRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService) // NEW
	collabHandler := handlers.NewCollaborationHandler(collabService)   // Collaboration 🤝
	adminHandler := handlers.NewAdminHandler(database)                 // Admin 👑
	referralHandler := handlers.NewReferralHandler(referralService)    // Referral System 🎁
	flightHandler := handlers.NewFlightHandler(flightGuardianService)  // Flight Guardian ✈️

	// Miru Chat (RAG) 💬
	chatService := services.NewChatService(tripRepo, cfg.OpenAIKey)
	chatHandler := handlers.NewChatHandler(chatService)

	// 6. Router
	r := http.SetupRouter(tripHandler, fbHandler, subHandler, webhookHandler, discoveryHandler, prefHandler, analyticsHandler, collabHandler, adminHandler, referralHandler, flightHandler, chatHandler, cfg.AllowOrigins, cfg.ClerkSecretKey)

	// 6.5 Flight Guardian Scheduler ✈️
	flightScheduler := scheduler.NewFlightGuardianScheduler(flightGuardianService)
	if err := flightScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start Flight Guardian scheduler: %v", err)
	}
	defer flightScheduler.Stop()

	// 7. Run
	log.Printf("Server running on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
