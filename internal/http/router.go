package http

import (
	"log"
	"strings"
	"time"
	"travelmate/internal/http/handlers"
	"travelmate/internal/http/middleware"
	"travelmate/internal/landmark"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	tripHandler *handlers.TripHandler,
	fbHandler *handlers.FeedbackHandler,
	subHandler *handlers.SubscriptionHandler,
	webhookHandler *handlers.WebhookHandler,
	discoveryHandler *handlers.DiscoveryHandler,
	prefHandler *handlers.PreferencesHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	collabHandler *handlers.CollaborationHandler,
	adminHandler *handlers.AdminHandler,                     // Admin 👑
	referralHandler *handlers.ReferralHandler,               // Referral System 🎁
	flightHandler *handlers.FlightHandler,                   // Flight Guardian ✈️
	chatHandler *handlers.ChatHandler,                       // Miru Chat (RAG) 💬
	knowledgeHandler *handlers.KnowledgeHandler,             // Local Knowledge (RAG) 🧠
	featureInterestHandler *handlers.FeatureInterestHandler, // Feature Interest 🔔
	passportHandler *handlers.PassportHandler,               // Digital Passport 🛂
	radarHandler *handlers.RadarHandler,                     // Miru Radar 📡
	landmarkHandler *landmark.Handler,                       // Landmark Domain 🏛️
	allowOrigins string,
	clerkKey string,
	userEmailSyncer middleware.UserSyncer, // User DB sync 📧
) *gin.Engine {

	r := gin.New()

	// Middleware Standar
	r.Use(gin.Recovery())
	r.Use(middleware.JSONLogger())

	// 🛠️ CONFIG CORS
	if allowOrigins == "" || allowOrigins == "*" {
		log.Fatal("ALLOW_ORIGINS environment variable must be set. Refusing to start with wildcard CORS.")
	}
	origins := strings.Split(allowOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	// Logging allowed origins for Railway debugging
	log.Println("🌐 CORS Allowed Origins:", origins)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-User-ID", "X-Mayar-Signature", "X-Admin-Secret"},
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

			// 1. Trip Core (Optional Auth for Registered User context)
			publicTrips := v1.Group("/")
			publicTrips.Use(middleware.OptionalAuthMiddleware(clerkKey))
			{
				publicTrips.POST("/trips", tripHandler.CreateTripAsync)
				publicTrips.POST("/trips/stream", tripHandler.CreateTripStream)
				publicTrips.POST("/trips/generate/stream", tripHandler.CreateTripSSE)
				publicTrips.GET("/trips/:id", tripHandler.GetTrip)
				publicTrips.GET("/trips/:id/enrich/:day_index/:activity_index", tripHandler.EnrichActivity)
				publicTrips.POST("/trips/:id/feedback", fbHandler.SubmitFeedback)
				publicTrips.GET("/radar", radarHandler.GetRadar) // Miru Radar 📡
			}

			// 1.5 Landmark Images 🏛️ (public — no auth, cached on disk)
			v1.GET("/landmarks/:slug/:variant", landmarkHandler.GetLandmark)

			// 2. Discovery & Inspiration 🚀
			v1.GET("/discovery", tripHandler.GetDiscovery)
			v1.GET("/discovery/trending", discoveryHandler.GetTrending)
			v1.GET("/discovery/explore", discoveryHandler.GetExplore)

			// 2.1 Discovery Teaser — RAG local insights (no auth, no OpenAI) ✨
			v1.GET("/destinations/:name/insights", knowledgeHandler.GetInsights)

			// 3. Utilities
			v1.POST("/alternatives", tripHandler.GetAlternatives)

			// 4. Webhooks (Public)
			// Stripe webhook disabled — Mayar.id is the active payment provider.
			// api.POST("/webhooks/stripe", webhookHandler.HandleStripeWebhook)
			api.POST("/webhooks/mayar", webhookHandler.HandleMayarWebhook)

			// 🌐 PUBLIC SHARE ROUTES (No Auth — for shareable trip links & OG crawlers)
			publicShare := v1.Group("/public")
			{
				publicShare.GET("/trips/:id", tripHandler.GetPublicTrip)
			}

			// 🔒 PROTECTED ROUTES (Requires Clerk Authentication)
			// ============================================================
			protected := v1.Group("/")
			protected.Use(middleware.AuthMiddleware(clerkKey, userEmailSyncer))
			{
				protected.GET("/trips", tripHandler.ListTrips)
				protected.POST("/trips/save", tripHandler.SaveTrip)
				protected.DELETE("/trips/:id", tripHandler.DeleteTrip)
				protected.POST("/trips/:id/travel-mode", tripHandler.ActivateTravelMode)
				protected.POST("/trips/:id/logistics/transport", tripHandler.GenerateTransportOnDemand)
				protected.POST("/trips/:id/refine", tripHandler.RefineTrip) // Miru AI Assistant 🧠
				protected.GET("/trips/:id/alternatives/:day_index/:activity_index", tripHandler.GetActivityAlternativesByIndex)
				protected.POST("/trips/:id/swap/:day_index/:activity_index", tripHandler.SwapActivity)
				protected.POST("/trips/:id/activities", tripHandler.AddActivity)
				protected.GET("/trips/:id/suggestions/:day_index", tripHandler.GetAddActivitySuggestions)
				protected.DELETE("/trips/:id/activities/:day_index/:activity_index", tripHandler.DeleteActivity)
				protected.GET("/trips/:id/export/pdf", tripHandler.ExportPDF)         // Premium Export 📄
				protected.GET("/trips/:id/accommodation", tripHandler.GetAccommodation) // Accommodation recs

				// 4. Subscription
				protected.GET("/user/subscription", subHandler.GetSubscription)
				protected.GET("/user/quota", subHandler.GetQuota)
				protected.POST("/user/subscription/checkout", subHandler.CreateCheckoutSession)

				// 5. User Preferences (Travel DNA) 🧬
				protected.GET("/user/preferences", prefHandler.GetPreferences)
				protected.PUT("/user/preferences", prefHandler.UpdatePreferences)

				// 6. Analytics & Impact 📈
				protected.POST("/analytics/events", analyticsHandler.TrackEvent)
				protected.GET("/analytics/impact", analyticsHandler.GetImpactStats)

				// 7. Collaboration 🤝
				protected.GET("/trips/:id/collaborators", collabHandler.GetCollaborators)
				protected.POST("/trips/:id/invite", collabHandler.InviteCollaborator)
				protected.DELETE("/trips/:id/collaborators/:userId", collabHandler.RemoveCollaborator)
				protected.PUT("/trips/:id/collaborators/:userId", collabHandler.UpdateCollaboratorRole)

				// 8. Referral System 🎁
				protected.POST("/referrals/claim", referralHandler.ClaimReferral)
				protected.GET("/user/referral", referralHandler.GetReferralInfo)
				protected.GET("/referrals/rank", referralHandler.GetUserRank) // My Rank 🏅

				// 8.1 Gamification (Phase 3) 🏆
				protected.GET("/referrals/leaderboard", referralHandler.GetLeaderboard)
				protected.GET("/user/achievements", referralHandler.GetUserAchievements)
				protected.GET("/users/me/achievement-progress", referralHandler.GetAchievementProgress)

				// 9. Flight Guardian ✈️
				protected.POST("/trips/:id/track-flights", flightHandler.TrackFlight)
				protected.GET("/trips/:id/alerts", flightHandler.GetTripAlerts)
				protected.DELETE("/alerts/:id", flightHandler.DeactivateAlert)
				protected.GET("/flights/locations", flightHandler.SearchLocations) // New: Airport Autocomplete
				protected.GET("/flights/search", flightHandler.SearchFlightOffers) // New: Flight Search

				// 10. Miru Chat (RAG) 💬
				protected.POST("/chat/completion", chatHandler.ChatCompletion)
				protected.GET("/chat/usage", chatHandler.GetChatUsage)

				// 11. Feature Interest (Notify me) 🔔
				protected.POST("/feature-interest", featureInterestHandler.NotifyInterest)

				// 12. Digital Passport 🛂
				protected.GET("/passport", passportHandler.GetUserStamps)
				protected.POST("/passport/claim", passportHandler.ClaimStamp)
				protected.GET("/passport/check", passportHandler.CheckStamp)
			}

			// 10. Admin Dashboard 👑
			admin := v1.Group("/admin")
			admin.Use(middleware.AdminAuthMiddleware())
			{
				admin.GET("/stats", adminHandler.GetStats)
				admin.GET("/users", adminHandler.GetUsers)
				admin.POST("/users/:userId/subscription", adminHandler.SetSubscription)

				// RAG: Local Knowledge Ingestion 🧠
				admin.POST("/knowledge", knowledgeHandler.IngestKnowledge)

				// Landmark Batch Seed 🏛️
				admin.POST("/landmarks/seed", landmarkHandler.SeedLandmarks)
			}
		}
	}

	return r
}
