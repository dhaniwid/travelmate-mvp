package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"travelmate/internal/domain"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type TripHandler struct {
	Service *services.TripService
}

func NewTripHandler(s *services.TripService) *TripHandler {
	return &TripHandler{Service: s}
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
			return false
		case <-c.Request.Context().Done():
			return false
		}
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
	var req domain.Trip

	// 1. Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "validation_error",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 2. Validasi ID User (Wajib ada dari Clerk)
	if req.UserID == "" {
		c.JSON(http.StatusUnauthorized, domain.APIError{
			Code:    "unauthorized",
			Message: "User ID is missing",
		})
		return
	}

	// FORCE the status to UPCOMING when the user explicitly saves it
	req.Status = "UPCOMING"

	// 3. Panggil Service
	if err := h.Service.SaveUserTrip(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: "Failed to save trip: " + err.Error(),
		})
		return
	}

	// 4. Response Sukses
	c.JSON(http.StatusCreated, gin.H{
		"message": "Trip confirmed! Countdown started. ⏳",
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
