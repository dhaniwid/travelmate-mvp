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

	// 🛠️ CONFIG CORS OPTIMIZED FOR SSE & CLERK
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // Di production, ganti dengan domain frontend Anda
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		// Tambahkan "Authorization" agar token Clerk bisa lewat
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length", "Connection", "Cache-Control", "Transfer-Encoding"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Grouping Routes
	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			// ============================================================
			// 🌍 PUBLIC ROUTES (Accessible by Guest/Anonymous)
			// ============================================================

			// Trip Generation (Streaming) - Kita biarkan public agar guest bisa mencoba
			// User ID akan dikirim opsional di body jika user login
			v1.POST("/trips/stream", tripHandler.CreateTripStream)

			// Get Single Trip (Detail) - Public agar bisa dishare link-nya ke teman
			v1.GET("/trips/:id", tripHandler.GetTrip)

			// Utilities
			v1.POST("/alternatives", tripHandler.GetAlternatives)
			v1.POST("/trips/:id/feedback", fbHandler.SubmitFeedback)

			// ============================================================
			// 🔒 PROTECTED ROUTES (Requires Clerk Authentication)
			// ============================================================
			protected := v1.Group("/")
			// Pasang Middleware Auth di group ini
			protected.Use(middleware.AuthMiddleware())
			{
				// 1. History (List Trips)
				// Wajib login karena memfilter berdasarkan user_id dari token
				protected.GET("/trips", tripHandler.ListTrips)

				// 2. Save/Confirm Trip
				// Wajib login karena user anonim tidak bisa menyimpan ke history permanen
				protected.POST("/trips/save", tripHandler.SaveTrip)

				// 3. Delete Trip
				// Wajib login untuk keamanan
				protected.DELETE("/trips/:id", tripHandler.DeleteTrip)
			}
		}
	}

	return r
}
