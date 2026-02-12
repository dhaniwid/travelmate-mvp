package handlers

import (
	"net/http"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"

	"github.com/gin-gonic/gin"
)

type PreferencesHandler struct {
	Repo *repositories.PreferencesRepository
}

func NewPreferencesHandler(repo *repositories.PreferencesRepository) *PreferencesHandler {
	return &PreferencesHandler{Repo: repo}
}

// GetPreferences handles GET /api/v1/user/preferences
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	// Get UserID from context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	prefs, err := h.Repo.GetPreferences(c, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences"})
		return
	}

	// If no preferences found, return default empty structure or specific message
	// For frontend convenience, we return a default structure even if it's new
	if prefs == nil {
		c.JSON(http.StatusOK, domain.UserPreferences{
			UserID:      userID.(string),
			Pace:        "BALANCED",
			BudgetTier:  "MID",
			Dietary:     []string{},
			Interests:   []string{},
			TravelStyle: []string{},
		})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences handles PUT /api/v1/user/preferences
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input domain.UserPreferences
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Override UserID from context to prevent ID spoofing
	input.UserID = userID.(string)

	if err := h.Repo.UpsertPreferences(c, &input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully", "data": input})
}
