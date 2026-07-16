package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
)

type TripHandler struct {
	Service    ITripService
	SubService ISubscriptionService
	CollabRepo ICollaboratorRepository
}

func NewTripHandler(s ITripService, sub ISubscriptionService, collabRepo ICollaboratorRepository) *TripHandler {
	return &TripHandler{Service: s, SubService: sub, CollabRepo: collabRepo}
}

type AlternativesRequest struct {
	Destination      string   `json:"destination" binding:"required"`
	OriginalActivity string   `json:"original_activity" binding:"required"`
	Location         string   `json:"location" binding:"required"` // Area sekitar (misal: "Ubud, Bali")
	Tags             []string `json:"tags"`                        // e.g. ["Nature", "Quiet"]
}

// CreateTripStream (Modern/Parallel Streaming)
// Dioptimasi untuk pembacaan chunk-by-chunk di Frontend (SSE)
func (h *TripHandler) CreateTripStream(c *gin.Context) {
	var req domain.Trip
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Set Header SSE agar browser menjaga koneksi tetap terbuka
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	eventChan := make(chan string)
	doneChan := make(chan bool)

	log.Printf("Creating trip for UserID: %s", req.UserID)

	// Duration gate: FREE users limited to 3 days max
	if req.UserID != "" && req.UserID != "guest" && req.TripDays > 3 {
		isPro, err := h.SubService.IsPROUser(c.Request.Context(), req.UserID)
		if err != nil {
			log.Printf("IsPROUser check error: %v", err)
		} else if !isPro {
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "free_tier_limit",
				Message: "Trip lebih dari 3 hari tersedia di PRO",
			})
			return
		}
	}

	// Jalankan streaming di Service
	go h.Service.GenerateTripStream(c.Request.Context(), req, eventChan, doneChan)

	// Flush data ke client secara real-time
	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-eventChan:
			// Kirim raw JSON diikuti newline agar mudah di-parse fetch reader di Frontend
			fmt.Fprintf(w, "%s\n", event)
			return true
		case <-doneChan:
			// Send final "done" event before closing
			doneEvent := `{"type":"done","data":{}}`
			fmt.Fprintf(w, "%s\n", doneEvent)
			return false
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// CreateTripAsync handles the initial fast generation and returns immediately (M-123)
func (h *TripHandler) CreateTripAsync(c *gin.Context) {
	var req domain.Trip
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Duration gate: FREE users limited to 3 days max
	if req.UserID != "" && req.UserID != "guest" && req.TripDays > 3 {
		isPro, err := h.SubService.IsPROUser(c.Request.Context(), req.UserID)
		if err != nil {
			log.Printf("IsPROUser check error: %v", err)
		} else if !isPro {
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "free_tier_limit",
				Message: "Trip lebih dari 3 hari tersedia di PRO",
			})
			return
		}
	}

	// Call Service
	trip, err := h.Service.GenerateTripAsync(c.Request.Context(), req)
	if err != nil {
		log.Printf("❌ [TRIPS HANDLER] GenerateTripAsync error: %v", err)
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "generation_error",
			Message: err.Error(),
		})
		return
	}

	// Return Success (Immediate 200)
	c.JSON(http.StatusOK, gin.H{
		"trip_id":           trip.ID,
		"enrichment_status": trip.EnrichmentStatus,
		"itinerary_status":  trip.ItineraryStatus,
		"message":           "Trip generation started. Phase 1 (Overview) complete. Phase 2 (Detailed) is running in background.",
	})
}

// GetTrip (Detail Trip)
func (h *TripHandler) GetTrip(c *gin.Context) {
	id := c.Param("id")
	result, err := h.Service.GetTrip(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: err.Error(),
		})
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, domain.APIError{Code: "not_found", Message: "Trip not found"})
		return
	}

	// 🛡️ SECURITY: IDOR CHECK (Owner OR Collaborator)
	// Allow access if trip is guest/public OR user is owner OR user is a collaborator
	if result.Trip.UserID != "" && result.Trip.UserID != "guest" {
		userID := c.GetString("userID")

		// First check: Is user the owner?
		isOwner := result.Trip.UserID == userID

		// Second check: Is user a collaborator?
		hasCollabAccess := false
		if !isOwner && h.CollabRepo != nil {
			var err error
			hasCollabAccess, err = h.CollabRepo.HasAccess(c.Request.Context(), id, userID)
			if err != nil {
				fmt.Printf("⚠️ Collaboration access check failed for TripID=%s, UserID=%s: %v\n", id, userID, err)
				// On error, we default to deny access (fail-safe)
				hasCollabAccess = false
			}
		}

		// Deny access if user is neither owner nor collaborator
		if !isOwner && !hasCollabAccess {
			fmt.Printf("\n🚨 ACCESS DENIED: TripID=%s | OwnerID=%s | RequestorID=%s | IsOwner=%v | HasCollabAccess=%v\n",
				id, result.Trip.UserID, userID, isOwner, hasCollabAccess)
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "forbidden",
				Message: "You do not have permission to view this trip",
			})
			return
		}

		fmt.Printf("✅ ACCESS GRANTED: TripID=%s | RequestorID=%s | IsOwner=%v | HasCollabAccess=%v\n",
			id, userID, isOwner, hasCollabAccess)
	}

	// Hidden gems are gated pending content curation (Phase 2) — strip AI-generated content
	result.Plan.HiddenGem = nil

	c.JSON(http.StatusOK, result)
}

// GetPublicTrip handles GET /api/v1/public/trips/:id
// Unauthenticated read-only endpoint for public share links.
// Returns the same trip data as GetTrip but skips the IDOR ownership check.
func (h *TripHandler) GetPublicTrip(c *gin.Context) {
	id := c.Param("id")
	result, err := h.Service.GetTrip(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: err.Error(),
		})
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, domain.APIError{
			Code:    "not_found",
			Message: "Trip not found",
		})
		return
	}

	// Strip sensitive fields before returning
	result.Trip.UserID = ""

	c.JSON(http.StatusOK, result)
}

// ListTrips (History)
func (h *TripHandler) ListTrips(c *gin.Context) {
	// 1. AMBIL USER ID DARI CONTEXT (Middleware Clerk)
	// Pastikan middleware auth Anda men-set "user_id" ke context
	userID := c.GetString("userID")

	// 🛡️ SECURITY GUARD: Jika tidak ada user_id, tolak akses
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{
			Code:    "unauthorized",
			Message: "You must be logged in to view history.",
		})
		return
	}

	// 2. PANGGIL SERVICE DENGAN USER ID
	trips, err := h.Service.GetUserTrips(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: "Failed to fetch history: " + err.Error(),
		})
		return
	}

	// 3. HANDLE EMPTY STATE
	// Kembalikan array kosong [] jika null, agar frontend mudah mapping
	if trips == nil {
		trips = []domain.Trip{}
	}

	c.JSON(http.StatusOK, gin.H{"data": trips})
}

// SaveTrip menangani POST /api/trips
func (h *TripHandler) SaveTrip(c *gin.Context) {
	// Kita bisa buat struct request kecil agar tidak perlu bind seluruh object Trip yang besar
	// karena kita hanya butuh ID dan UserID.
	var req struct {
		ID       string           `json:"id"`
		UserID   string           `json:"user_id"`
		PlanData *domain.TripPlan `json:"plan_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "validation_error",
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 🛡️ SECURITY: ALWAYS USE AUTHENTICATED USER ID FROM CONTEXT
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User ID missing from context"})
		return
	}

	// Buat object domain trip dummy hanya untuk passing data ke service
	trip := &domain.Trip{
		ID:       req.ID,
		UserID:   userID, // Use context userID, not the one from request body
		PlanData: req.PlanData,
	}

	// Panggil Service
	if err := h.Service.SaveUserTrip(c.Request.Context(), trip); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: "Failed to claim trip: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Trip successfully saved to your history! 🚀",
		"trip_id": req.ID,
		"status":  "UPCOMING",
	})
}

// ActivateTravelMode handles POST /api/v1/trips/:id/travel-mode
func (h *TripHandler) ActivateTravelMode(c *gin.Context) {
	tripID := c.Param("id")
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "Missing User Context"})
		return
	}
	if err := h.Service.ActivateTravelMode(c.Request.Context(), tripID, userID); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "travel_mode_active": true})
}

// GenerateTransportOnDemand POST /trips/:id/logistics/transport (MT-79)
func (h *TripHandler) GenerateTransportOnDemand(c *gin.Context) {
	tripID := c.Param("id")
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "Missing User Context"})
		return
	}

	var req struct {
		OriginCity string `json:"origin_city" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: "origin_city is required"})
		return
	}

	options, err := h.Service.GenerateTransportOnDemand(c.Request.Context(), tripID, req.OriginCity, userID)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, domain.APIError{Code: "forbidden", Message: "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "generation_failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "transport_options": options})
}

// DeleteTrip menangani DELETE /api/v1/trips/:id
func (h *TripHandler) DeleteTrip(c *gin.Context) {
	tripID := c.Param("id")

	// 🛡️ SECURITY: ALWAYS USE AUTHENTICATED USER ID FROM CONTEXT (Fix IDOR)
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "Missing User Context"})
		return
	}

	err := h.Service.DeleteUserTrip(c.Request.Context(), tripID, userID)
	if err != nil {
		// Bedakan error not found vs internal
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "delete_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Trip deleted successfully"})
}

func (h *TripHandler) GetAlternatives(c *gin.Context) {
	var req AlternativesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: err.Error()})
		return
	}

	// Panggil Service
	results, err := h.Service.GetActivityAlternatives(
		c.Request.Context(),
		req.Destination,
		req.OriginalActivity,
		req.Location,
		req.Tags,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "ai_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

// GetPackingList menangani GET /api/v1/trips/:id/packing-list
func (h *TripHandler) GetPackingList(c *gin.Context) {
	tripID := c.Param("id")

	list, err := h.Service.GetPackingList(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Endpoint: GET /api/v1/discovery?city=Surabaya
func (h *TripHandler) GetDiscovery(c *gin.Context) {
	// 1. Ambil Query Param
	city := c.Query("city")

	// 2. Validasi Input
	if city == "" {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "bad_request",
			Message: "City parameter is required",
		})
		return
	}

	// 3. Panggil Service
	data, err := h.Service.GetDestinationDiscovery(c.Request.Context(), city)
	if err != nil {
		// Log error jika perlu
		// log.Printf("Discovery Error: %v", err)

		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: "Failed to get discovery info: " + err.Error(),
		})
		return
	}

	// 4. Return Success JSON
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// Sub-struct for Refinement
type RefineTripRequest struct {
	Instruction string `json:"instruction" binding:"required"`
}

// RefineTrip Request Adjustment via Chat
func (h *TripHandler) RefineTrip(c *gin.Context) {
	tripID := c.Param("id")
	var req RefineTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: err.Error()})
		return
	}

	updatedPlan, err := h.Service.RefineTrip(c.Request.Context(), tripID, req.Instruction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "Refinement Failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedPlan})
}

// ExportPDF handles GET /trips/:id/export/pdf
func (h *TripHandler) ExportPDF(c *gin.Context) {
	tripID := c.Param("id")

	// 0. CHECK PREMIUM STATUS (PRO ONLY)
	userID := c.GetString("userID") // From Auth Middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User not authenticated"})
		return
	}

	user, err := h.SubService.GetUserSubscription(c.Request.Context(), userID, "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: "Failed to verify subscription"})
		return
	}

	if user.SubscriptionTier != "PRO" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":        "premium_feature",
			"message":      "PDF Export is a specific feature for Miru PRO members. Upgrade to unlock magazine-style exports!",
			"current_tier": user.SubscriptionTier,
		})
		return
	}

	// 1. Call Service
	pdfBytes, filename, err := h.Service.ExportTripToPDF(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "pdf_generation_failed", Message: err.Error()})
		return
	}

	// 2. Set Headers for Download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	// 3. Write Data
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// GetAccommodation handles GET /api/v1/trips/:id/accommodation
// Returns the AI-generated accommodation recommendations for a trip.
// Auth: same IDOR + collaborator check as GetTrip.
func (h *TripHandler) GetAccommodation(c *gin.Context) {
	tripID := c.Param("id")
	userID := c.GetString("userID")

	result, err := h.Service.GetTrip(c.Request.Context(), tripID)
	if err != nil || result == nil {
		c.JSON(http.StatusNotFound, domain.APIError{Code: "not_found", Message: "Trip not found"})
		return
	}

	// Mirror the full auth check from GetTrip: owner OR collaborator.
	if result.Trip.UserID != "" && result.Trip.UserID != "guest" {
		isOwner := result.Trip.UserID == userID
		hasCollabAccess := false
		if !isOwner && h.CollabRepo != nil {
			hasCollabAccess, _ = h.CollabRepo.HasAccess(c.Request.Context(), tripID, userID)
		}
		if !isOwner && !hasCollabAccess {
			c.JSON(http.StatusForbidden, domain.APIError{Code: "forbidden", Message: "Access denied"})
			return
		}
	}

	options := result.Plan.AccommodationOptions
	if options == nil {
		options = []domain.AccommodationOption{}
	}

	c.JSON(http.StatusOK, gin.H{
		"trip_id":  tripID,
		"options":  options,
		"is_ready": len(options) > 0,
	})
}

// EnrichActivity handles GET /api/v1/trips/:id/enrich/:day_index/:activity_index
func (h *TripHandler) EnrichActivity(c *gin.Context) {
	tripID := c.Param("id")
	dayIdxStr := c.Param("day_index")
	actIdxStr := c.Param("activity_index")

	dayIdx, err := strconv.Atoi(dayIdxStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: "Invalid day index"})
		return
	}

	actIdx, err := strconv.Atoi(actIdxStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: "Invalid activity index"})
		return
	}

	enrichedActivity, err := h.Service.EnrichActivity(c.Request.Context(), tripID, dayIdx, actIdx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "enrichment_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": enrichedActivity,
	})
}

// GetActivityAlternativesByIndex handles GET /api/v1/trips/:id/alternatives/:day_index/:activity_index
func (h *TripHandler) GetActivityAlternativesByIndex(c *gin.Context) {
	tripID := c.Param("id")
	dayIdx, _ := strconv.Atoi(c.Param("day_index"))
	actIdx, _ := strconv.Atoi(c.Param("activity_index"))

	refresh := c.Query("refresh") == "true"
	results, err := h.Service.GetActivityAlternativesByIndex(c.Request.Context(), tripID, dayIdx, actIdx, refresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

type SwapActivityRequest struct {
	Alternative domain.ActivityAlternative `json:"alternative" binding:"required"`
}

// SwapActivity handles POST /api/v1/trips/:id/swap/:day_index/:activity_index
func (h *TripHandler) SwapActivity(c *gin.Context) {
	tripID := c.Param("id")
	dayIdx, _ := strconv.Atoi(c.Param("day_index"))
	actIdx, _ := strconv.Atoi(c.Param("activity_index"))

	var req SwapActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: err.Error()})
		return
	}

	err := h.Service.SwapActivity(c.Request.Context(), tripID, dayIdx, actIdx, req.Alternative)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Activity swapped successfully"})
}

type AddActivityRequest struct {
	DayIdx      int    `json:"day_index"`
	Title       string `json:"title" binding:"required"`
	Time        string `json:"time" binding:"required"`
	AutoEnhance bool   `json:"auto_enhance"`
}

// AddActivity handles POST /api/v1/trips/:id/activities
func (h *TripHandler) AddActivity(c *gin.Context) {
	tripID := c.Param("id")
	var req AddActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: err.Error()})
		return
	}

	updatedPlan, err := h.Service.AddActivity(c.Request.Context(), tripID, req.DayIdx, req.Title, req.Time, req.AutoEnhance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedPlan})
}

// GetAddActivitySuggestions handles GET /api/v1/trips/:id/suggestions/:day_index?time=HH:MM
func (h *TripHandler) GetAddActivitySuggestions(c *gin.Context) {
	tripID := c.Param("id")
	dayIdx, _ := strconv.Atoi(c.Param("day_index"))
	timeStr := c.Query("time")

	suggestions, err := h.Service.GetAddActivitySuggestions(c.Request.Context(), tripID, dayIdx, timeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": suggestions})
}

// CreateTripSSE handles POST /api/v1/trips/generate/stream
// Inserts a trip stub immediately, emits "trip_created" SSE so the frontend can navigate
// within ~1–2s, then streams AI generation and emits "skeleton_complete" when done.
func (h *TripHandler) CreateTripSSE(c *gin.Context) {
	var req domain.Trip
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: err.Error()})
		return
	}

	// Duration gate: FREE users limited to 3 days max
	if req.UserID != "" && req.UserID != "guest" && req.TripDays > 3 {
		isPro, err := h.SubService.IsPROUser(c.Request.Context(), req.UserID)
		if err != nil {
			log.Printf("IsPROUser check error: %v", err)
		} else if !isPro {
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "free_tier_limit",
				Message: "Trip lebih dari 3 hari tersedia di PRO",
			})
			return
		}
	}

	// SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	events := make(chan string, 8)

	// Run generation in a goroutine; closes events when done
	go func() {
		defer close(events)
		h.Service.GenerateTripSSE(c.Request.Context(), req, events)
	}()

	// Stream events to client
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				return false // channel closed
			}
			fmt.Fprint(w, event)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// DeleteActivity handles DELETE /api/v1/trips/:id/activities/:day_index/:activity_index
func (h *TripHandler) DeleteActivity(c *gin.Context) {
	tripID := c.Param("id")
	dayIdx, _ := strconv.Atoi(c.Param("day_index"))
	actIdx, _ := strconv.Atoi(c.Param("activity_index"))

	updatedPlan, err := h.Service.DeleteActivity(c.Request.Context(), tripID, dayIdx, actIdx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedPlan})
}
