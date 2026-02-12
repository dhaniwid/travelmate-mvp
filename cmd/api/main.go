package main

import (
	"log"
	"travelmate/internal/config"
	"travelmate/internal/db"
	"travelmate/internal/http"
	"travelmate/internal/http/handlers"
	"travelmate/internal/repositories"
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
	prefRepo := repositories.NewPreferencesRepository(database) // NEW

	// 4. Services (Dependency Injection)
	promptService := services.NewPromptService(database)
	imageSvc := services.NewImageService(cfg.GoogleAPIKey, cfg.GoogleCXId)
	pdfSvc := services.NewPDFService() // NEW

	// Stripe Client
	stripeClient := stripePkg.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookKey)

	subService := services.NewSubscriptionService(userRepo, subRepo, stripeClient)
	discoveryService := services.NewDiscoveryService(destRepo)

	var plannerEngine services.PlannerEngine
	if cfg.OpenAIKey != "" {
		plannerEngine = services.NewAIPlanner(cfg.OpenAIKey, promptService, prefRepo)
	} else {
		plannerEngine = services.NewTemplatePlanner()
	}

	// 4.3 Location Service needs promptService
	locationService := services.NewLocationService(locRepo, promptService, cfg.OpenAIKey, imageSvc)

	// Enrichment Service
	enrichService := services.NewEnrichmentService(tripRepo, cfg.GoogleAPIKey)

	transportService := services.NewTransportService(transportRepo)

	tripService := services.NewTripService(tripRepo, fbRepo, accommodationRepo, attractionRepo, transportRepo,
		perfRepo, discoveryRepo, plannerEngine, locationService, transportService, imageSvc, pdfSvc, enrichService)

	// 5. Handlers
	tripHandler := handlers.NewTripHandler(tripService, subService)
	fbHandler := handlers.NewFeedbackHandler(tripService)
	subHandler := handlers.NewSubscriptionHandler(subService)
	webhookHandler := handlers.NewWebhookHandler(subService, stripeClient)
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)
	prefHandler := handlers.NewPreferencesHandler(prefRepo) // NEW

	// 6. Router
	r := http.SetupRouter(tripHandler, fbHandler, subHandler, webhookHandler, discoveryHandler, prefHandler)

	// 7. Run
	log.Printf("Server running on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
