package handlers

import (
	"net/http"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	service *services.AnalyticsService
}

func NewAnalyticsHandler(service *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

type TrackEventRequest struct {
	EventType string                 `json:"event_type" binding:"required"`
	EventData map[string]interface{} `json:"event_data"`
}

// TrackEvent mencatat event dari frontend
func (h *AnalyticsHandler) TrackEvent(c *gin.Context) {
	var req TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		userID = "anonymous"
	}

	err := h.service.TrackEvent(c.Request.Context(), userID, req.EventType, req.EventData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "captured"})
}

// GetImpactStats mengambil statistik dampak user
func (h *AnalyticsHandler) GetImpactStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := h.service.GetImpactStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
