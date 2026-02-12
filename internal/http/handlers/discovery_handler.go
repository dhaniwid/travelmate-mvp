package handlers

import (
	"net/http"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type DiscoveryHandler struct {
	service *services.DiscoveryService
}

func NewDiscoveryHandler(service *services.DiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{service: service}
}

// GetTrending returns a list of trending destinations
// GET /api/v1/discovery/trending
func (h *DiscoveryHandler) GetTrending(c *gin.Context) {
	dests, err := h.service.GetTrendingDestinations(c.Request.Context(), 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trending destinations"})
		return
	}
	c.JSON(http.StatusOK, dests)
}

// GetExplore returns structured data for the explore page
// GET /api/v1/discovery/explore
func (h *DiscoveryHandler) GetExplore(c *gin.Context) {
	data, err := h.service.GetExploreData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch explore data"})
		return
	}
	c.JSON(http.StatusOK, data)
}
