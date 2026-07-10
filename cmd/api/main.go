package main

import (
	"log"
	"os"
	"travelmate/internal/config"
	"travelmate/internal/db"
	"travelmate/internal/http"
	"travelmate/internal/http/handlers"
	"travelmate/internal/landmark"
	"travelmate/internal/repositories"
	"travelmate/internal/scheduler"
	"travelmate/internal/services"
	stripePkg "travelmate/internal/stripe"

	// NOTE: stripe package retained — CreateCheckoutSession still active via /user/subscription/checkout.
	// TODO MIR-018: Replace with Mayar.id checkout flow and remove stripe package entirely.
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
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
	knowledgeRepo := repositories.NewKnowledgeRepository(database)     // Local Knowledge RAG 🧠
	featureInterestRepo := repositories.NewFeatureInterestRepository(database) // Feature Interest 🔔
	passportRepo := repositories.NewPassportRepository(database)              // Digital Passport 🛂

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
		plannerEngine = services.NewAIPlanner(cfg.OpenAIKey, promptService, prefRepo, amadeusService, knowledgeRepo) // Pass knowledgeRepo for RAG
	} else {
		plannerEngine = services.NewTemplatePlanner()
	}

	// 4.3 Location Service needs promptService
	locationService := services.NewLocationService(locRepo, promptService, cfg.OpenAIKey, imageSvc)

	// Enrichment Service
	enrichService := services.NewEnrichmentService(tripRepo, placeLibRepo, cfg.GoogleAPIKey)

	transportService := services.NewTransportService(transportRepo)
	passportService := services.NewPassportService(passportRepo) // Digital Passport 🛂
	radarService := services.NewRadarService(database)           // Miru Radar 📡

	tripService := services.NewTripService(tripRepo, fbRepo, accommodationRepo, attractionRepo, transportRepo,
		perfRepo, discoveryRepo, plannerEngine, locationService, transportService, imageSvc, pdfSvc, enrichService, subService, passportService)

	// 5. Handlers
	tripHandler := handlers.NewTripHandler(tripService, subService, collabRepo)
	fbHandler := handlers.NewFeedbackHandler(tripService)
	subHandler := handlers.NewSubscriptionHandler(subService)
	webhookHandler := handlers.NewWebhookHandler(subService)
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)
	prefHandler := handlers.NewPreferencesHandler(prefRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService) // NEW
	collabHandler := handlers.NewCollaborationHandler(collabService)   // Collaboration 🤝
	adminHandler := handlers.NewAdminHandler(database)                 // Admin 👑
	referralHandler := handlers.NewReferralHandler(referralService)    // Referral System 🎁
	flightHandler := handlers.NewFlightHandler(flightGuardianService)  // Flight Guardian ✈️

	// Miru Chat (RAG) 💬
	chatService := services.NewChatService(tripRepo, cfg.OpenAIKey)
	chatHandler := handlers.NewChatHandler(chatService, subService, database)

	// Local Knowledge Ingestion (RAG) 🧠 — shares the OpenAI client from plannerEngine
	openaiClient := openai.NewClient(cfg.OpenAIKey)
	knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeRepo, openaiClient)
	featureInterestHandler := handlers.NewFeatureInterestHandler(featureInterestRepo) // Feature Interest 🔔
	passportHandler := handlers.NewPassportHandler(passportService)
	radarHandler := handlers.NewRadarHandler(radarService) // Miru Radar 📡

	// Landmark Domain 🏛️
	landmarkBaseDir := os.Getenv("LANDMARK_BASE_DIR")
	if landmarkBaseDir == "" {
		landmarkBaseDir = "../travelmate-web/public/assets/landmarks"
	}
	landmarkRepo := landmark.NewRepo(database, landmarkBaseDir, "/assets/landmarks")
	landmarkSvc := landmark.NewService(landmarkRepo, openaiClient)
	landmarkHandler := landmark.NewHandler(landmarkSvc)

	// 6. Router
	r := http.SetupRouter(tripHandler, fbHandler, subHandler, webhookHandler, discoveryHandler, prefHandler, analyticsHandler, collabHandler, adminHandler, referralHandler, flightHandler, chatHandler, knowledgeHandler, featureInterestHandler, passportHandler, radarHandler, landmarkHandler, cfg.AllowOrigins, cfg.ClerkSecretKey, userRepo)

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
