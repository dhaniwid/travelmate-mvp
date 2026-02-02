package http

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
	"travelmate/internal/http/handlers"
	"travelmate/internal/http/middleware"
)

func SetupRouter(
	tripHandler *handlers.TripHandler,
	fbHandler *handlers.FeedbackHandler,
) *gin.Engine {

	r := gin.New()

	// Middleware Standar
	r.Use(gin.Recovery())
	r.Use(middleware.JSONLogger())

	// 🛠️ CONFIG CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
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

			// 1. Trip Core
			v1.POST("/trips/stream", tripHandler.CreateTripStream)
			v1.GET("/trips/:id", tripHandler.GetTrip)

			// 2. Discovery & Inspiration (NEW ROUTE) 🚀
			// Endpoint: GET /api/v1/discovery?city=Surabaya
			v1.GET("/discovery", tripHandler.GetDiscovery)

			// 3. Utilities & Feedback
			v1.POST("/alternatives", tripHandler.GetAlternatives)
			v1.POST("/trips/:id/feedback", fbHandler.SubmitFeedback)

			// ============================================================
			// 🔒 PROTECTED ROUTES (Requires Clerk Authentication)
			// ============================================================
			protected := v1.Group("/")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/trips", tripHandler.ListTrips)
				protected.POST("/trips/save", tripHandler.SaveTrip)
				protected.DELETE("/trips/:id", tripHandler.DeleteTrip)
			}
		}
	}

	return r
}
