package handlers

import (
	"context"
	"net/http"
	"strconv"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type IRadarService interface {
	GetRadar(ctx context.Context, lat, lng float64, radiusMeters int, userID string) (*services.RadarResponse, error)
}

type RadarHandler struct {
	Service IRadarService
}

func NewRadarHandler(svc IRadarService) *RadarHandler {
	return &RadarHandler{Service: svc}
}

// GET /api/v1/radar?lat=&lng=&radius=
func (h *RadarHandler) GetRadar(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng query params required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lng"})
		return
	}

	radius := 1000
	if r := c.Query("radius"); r != "" {
		if v, err := strconv.Atoi(r); err == nil && v > 0 {
			radius = v
		}
	}

	// userID from auth context — optional (radar works for guests too)
	userID := ""
	if uid, exists := c.Get("userID"); exists {
		userID = uid.(string)
	}

	result, err := h.Service.GetRadar(c.Request.Context(), lat, lng, radius, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": result})
}
