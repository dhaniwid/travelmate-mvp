package handlers

import (
	"fmt"
	"io"
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

// 1. CreateTrip (Legacy/Sequential)
// Digunakan jika Anda ingin response JSON utuh dalam satu kali request
func (h *TripHandler) CreateTrip(c *gin.Context) {
	var req domain.Trip
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    "validation_error",
			Message: err.Error(),
		})
		return
	}

	result, err := h.Service.GenerateAndSaveTrip(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// 2. CreateTripStream (Modern/Parallel Streaming)
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

// 3. GetTrip (Detail Trip)
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

// 4. ListTrips (History)
func (h *TripHandler) ListTrips(c *gin.Context) {
	trips, err := h.Service.ListTrips(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{
			Code:    "internal_error",
			Message: err.Error(),
		})
		return
	}

	// Kembalikan array kosong jika tidak ada data (bukan null)
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
		"message": "Trip saved successfully",
		"trip_id": req.ID,
	})
}
