package main

import (
	"log"
	"travelmate/internal/config"
	"travelmate/internal/db"
	"travelmate/internal/http"
	"travelmate/internal/http/handlers"
	"travelmate/internal/repositories"
	"travelmate/internal/services"

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

	// 4. Services (Dependency Injection)
	promptService := services.NewPromptService(database)
	imageSvc := services.NewImageService(cfg.GoogleAPIKey, cfg.GoogleCXId)

	var plannerEngine services.PlannerEngine
	if cfg.OpenAIKey != "" {
		plannerEngine = services.NewAIPlanner(cfg.OpenAIKey, promptService)
	} else {
		plannerEngine = services.NewTemplatePlanner()
	}

	// 4.3 Location Service needs promptService
	locationService := services.NewLocationService(locRepo, promptService, cfg.OpenAIKey, imageSvc)

	transportService := services.NewTransportService(transportRepo)

	tripService := services.NewTripService(tripRepo, fbRepo, accommodationRepo, attractionRepo, transportRepo,
		perfRepo, plannerEngine, locationService, transportService, imageSvc)

	// 5. Handlers
	tripHandler := handlers.NewTripHandler(tripService)
	fbHandler := handlers.NewFeedbackHandler(tripService)

	// 6. Router
	r := http.SetupRouter(tripHandler, fbHandler)

	// 7. Run
	log.Printf("Server running on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
