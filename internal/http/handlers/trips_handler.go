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
}

func NewTripHandler(s ITripService, sub ISubscriptionService) *TripHandler {
	return &TripHandler{Service: s, SubService: sub}
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

	// Quota Enforcement for Registered Users
	if req.UserID != "" && req.UserID != "guest" {
		allowed, err := h.SubService.CheckQuotaAvailability(c.Request.Context(), req.UserID)
		if err != nil {
			log.Printf("Quota check error: %v", err)
		} else if !allowed {
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "quota_exceeded",
				Message: "Monthly trip generation limit reached. Upgrade to PRO for unlimited planning!",
			})
			return
		}

		// Increment Quota
		if err := h.SubService.IncrementQuota(c.Request.Context(), req.UserID); err != nil {
			log.Printf("Failed to increment quota for %s: %v", req.UserID, err)
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

	// Quota Enforcement for Registered Users
	if req.UserID != "" && req.UserID != "guest" {
		allowed, err := h.SubService.CheckQuotaAvailability(c.Request.Context(), req.UserID)
		if err != nil {
			log.Printf("Quota check error: %v", err)
		} else if !allowed {
			c.JSON(http.StatusForbidden, domain.APIError{
				Code:    "quota_exceeded",
				Message: "Monthly trip generation limit reached. Upgrade to PRO for unlimited planning!",
			})
			return
		}

		// Increment Quota
		if err := h.SubService.IncrementQuota(c.Request.Context(), req.UserID); err != nil {
			log.Printf("Failed to increment quota for %s: %v", req.UserID, err)
		}
	}

	// Call Service
	trip, err := h.Service.GenerateTripAsync(c.Request.Context(), req)
	if err != nil {
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
		c.JSON(http.StatusNotFound, domain.APIError{
			Code:    "not_found",
			Message: "Trip not found",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListTrips (History)
func (h *TripHandler) ListTrips(c *gin.Context) {
	// 1. AMBIL USER ID DARI CONTEXT (Middleware Clerk)
	// Pastikan middleware auth Anda men-set "user_id" ke context
	userID := c.GetString("user_id")

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

	// Validasi User ID
	if req.UserID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User ID missing"})
		return
	}

	// Buat object domain trip dummy hanya untuk passing data ke service
	trip := &domain.Trip{
		ID:       req.ID,
		UserID:   req.UserID,
		PlanData: req.PlanData,
	}

	// 🛡️ SECURITY: CRITICAL QUOTA CHECK (Fix Bypass)
	// Check if user has quota BEFORE saving
	quota, err := h.SubService.GetUserQuota(c.Request.Context(), req.UserID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: "Quota check failed"})
		return
	}

	tripCount, err := h.Service.CountUserTrips(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: "Failed to verify trip count"})
		return
	}

	// Rule: If quota is restricted and limit reached
	if !quota.IsUnlimited && tripCount >= quota.QuotaLimit {
		c.JSON(http.StatusForbidden, gin.H{
			"error":        "quota_exceeded",
			"message":      fmt.Sprintf("You have reached your free account limit (%d trips). Upgrade to Pro for unlimited trips!", quota.QuotaLimit),
			"current_tier": "FREE",
			"trip_count":   tripCount,
			"limit":        quota.QuotaLimit,
		})
		return
	}

	// Panggil Service
	if err := h.Service.SaveUserTrip(c.Request.Context(), trip); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: "Failed to claim trip: " + err.Error(),
		})
		return
	}

	// 🛡️ SECURITY: INCREMENT QUOTA ON CLAIM
	// Since guest generation often bypasses the immediate increment (or uses guest bucket),
	// we must count the trip against the user's permanent quota here.
	if req.UserID != "guest" {
		if err := h.SubService.IncrementQuota(c.Request.Context(), req.UserID); err != nil {
			log.Printf("⚠️ [QUOTA] Failed to increment on claim for %s: %v", req.UserID, err)
		} else {
			log.Printf("📊 [QUOTA] Incremented for %s on trip claim", req.UserID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Trip successfully saved to your history! 🚀",
		"trip_id": req.ID,
		"status":  "UPCOMING",
	})
}

// DeleteTrip menangani DELETE /api/v1/trips/:id
func (h *TripHandler) DeleteTrip(c *gin.Context) {
	tripID := c.Param("id")

	// 1. Ambil User ID dari Token/Header (Asumsi kita kirim via Header 'X-User-Id' atau Auth Middleware)
	userID := c.GetHeader("X-User-ID")

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
	userID := c.GetString("user_id") // From Auth Middleware
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
