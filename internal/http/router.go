package http

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
	"travelmate/internal/http/handlers"
	"travelmate/internal/http/middleware"
)

func SetupRouter(tripHandler *handlers.TripHandler, fbHandler *handlers.FeedbackHandler) *gin.Engine {
	r := gin.New()

	// Middleware Standar
	r.Use(gin.Recovery())
	r.Use(middleware.JSONLogger())

	// 🛠️ CONFIG CORS OPTIMIZED FOR SSE
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // Di production, ganti dengan domain frontend Anda
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		// Penting untuk SSE: Izinkan browser membaca headers berikut
		ExposeHeaders:    []string{"Content-Length", "Connection", "Cache-Control", "Transfer-Encoding"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Grouping Routes
	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			// TRIP ROUTES
			// POST /api/v1/trips/stream -> Untuk fitur Parallel Streaming (SSE)
			v1.POST("/trips/stream", tripHandler.CreateTripStream)

			// POST /api/v1/trips -> Legacy/Non-streaming creation
			v1.POST("/trips", tripHandler.CreateTrip)

			// GET /api/v1/trips -> List semua trip
			v1.GET("/trips", tripHandler.ListTrips)

			// GET /api/v1/trips/:id -> Detail trip tunggal
			v1.GET("/trips/:id", tripHandler.GetTrip)

			// FEEDBACK ROUTES
			v1.POST("/trips/:id/feedback", fbHandler.SubmitFeedback)
		}
	}

	return r
}
