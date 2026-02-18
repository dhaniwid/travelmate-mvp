package handlers

import (
	"net/http"

	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

// FlightHandler handles flight-related HTTP requests
type FlightHandler struct {
	guardianService *services.FlightGuardianService
}

// NewFlightHandler creates a new flight handler
func NewFlightHandler(guardianService *services.FlightGuardianService) *FlightHandler {
	return &FlightHandler{
		guardianService: guardianService,
	}
}

// TrackFlightRequest represents the request body for activating Flight Guardian
type TrackFlightRequest struct {
	OriginAirport      string `json:"origin_airport" binding:"required"`
	DestinationAirport string `json:"destination_airport" binding:"required"`
}

// TrackFlight activates price monitoring for a trip
// POST /api/v1/trips/:id/track-flights
func (h *FlightHandler) TrackFlight(c *gin.Context) {
	tripID := c.Param("id")

	// Parse request body
	var req TrackFlightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origin_airport and destination_airport are required"})
		return
	}

	// Activate guardian
	status, err := h.guardianService.ActivateGuardian(c.Request.Context(), tripID, req.OriginAirport, req.DestinationAirport)
	if err != nil {
		// Check for specific errors
		if err.Error() == "trip not found: "+tripID {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "cannot track flights for past trips" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// General error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate Flight Guardian: " + err.Error()})
		return
	}

	// Return status
	c.JSON(http.StatusOK, status)
}

// GetTripAlerts retrieves all price alerts for a trip
// GET /api/v1/trips/:id/alerts
func (h *FlightHandler) GetTripAlerts(c *gin.Context) {
	tripID := c.Param("id")

	alerts, err := h.guardianService.GetTripAlerts(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// DeactivateAlert stops tracking a specific alert
// DELETE /api/v1/alerts/:id
func (h *FlightHandler) DeactivateAlert(c *gin.Context) {
	alertID := c.Param("id")

	if err := h.guardianService.DeactivateGuardian(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate alert: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "deactivated",
	})
}
// SearchLocations searches for airports/cities
// GET /api/v1/flights/locations?keyword=...
func (h *FlightHandler) SearchLocations(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyword is required"})
		return
	}

	locations, err := h.guardianService.SearchLocations(c.Request.Context(), keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search locations: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"locations": locations})
}

// SearchFlightOffers searches for flight offers
// GET /api/v1/flights/search?origin=...&dest=...&date=...&returnDate=...
func (h *FlightHandler) SearchFlightOffers(c *gin.Context) {
	origin := c.Query("origin")
	dest := c.Query("dest")
	date := c.Query("date")
	returnDate := c.Query("returnDate")

	if origin == "" || dest == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origin, dest, and date are required"})
		return
	}

	var returnDatePtr *string
	if returnDate != "" {
		returnDatePtr = &returnDate
	}

	offers, err := h.guardianService.SearchFlightOffers(c.Request.Context(), origin, dest, date, returnDatePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search flights: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}
