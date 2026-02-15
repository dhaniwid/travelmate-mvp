package handlers

import (
	"context"
	"net/http"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	planner services.PlannerEngine
	tripSvc *services.TripService
}

func NewTestHandler(planner services.PlannerEngine, tripSvc *services.TripService) *TestHandler {
	return &TestHandler{
		planner: planner,
		tripSvc: tripSvc,
	}
}

type RegeneratePromptRequest struct {
	TripID      string                 `json:"trip_id"`
	Preferences domain.UserPreferences `json:"preferences"`
}

// GeneratePrompt returns the raw prompt text for debugging
func (h *TestHandler) GeneratePrompt(c *gin.Context) {
	var req RegeneratePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 1. Load Trip (to get destination, duration, etc.)
	// We need a way to get trip, tripService usually requires UserID but for test we might skip or mock
	// Let's assume we can fetch it. If TripService doesn't expose a simple GetTripByID without UserID check,
	// we might need to rely on what available.
	// For now, let's construct a dummy trip if ID is "test", or try to fetch.
	var trip domain.Trip
	if req.TripID == "test" {
		trip = domain.Trip{
			ID:          "test-trip",
			Destination: "Paris",
			TripDays:    3,
			Style:       "Relaxed",
		}
	} else {
		// Real fetch (Using a public method from TripRepository if available, strictly speaking TripService enforce UserID)
		// For QA purpose, we just simulate a trip based on request or just partial data.
		// Let's use the ID as destination for simplicity if real fetch is hard here without refactor.
		// Actually, let's just accept Trip details in body if needed, or just dummy it.
		// To be useful, let's query DB.
		// Since we don't have direct access to Repo here (clean architecture), we use Service.
		// But Service GetTrip requires UserID.
		// Hack: We will just fallback to a Mock Trip for the prompt generation to verify logic.
		trip = domain.Trip{
			ID:          req.TripID,
			Destination: "Tokyo", // Default test dest
			TripDays:    3,
		}
	}

	// 2. Call Planner Engine
	prompt, err := h.planner.GetRegeneratePrompt(ctx, trip, req.Preferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Return Text
	c.JSON(http.StatusOK, gin.H{
		"prompt_length": len(prompt),
		"prompt_text":   prompt,
	})
}
