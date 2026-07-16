package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type IFeatureInterestRepo interface {
	SaveInterest(ctx context.Context, userID, featureKey string) error
}

type FeatureInterestHandler struct {
	Repo IFeatureInterestRepo
}

func NewFeatureInterestHandler(repo IFeatureInterestRepo) *FeatureInterestHandler {
	return &FeatureInterestHandler{Repo: repo}
}

func (h *FeatureInterestHandler) NotifyInterest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		req.FeatureKey = "hidden_gems"
	}

	if err := h.Repo.SaveInterest(c.Request.Context(), userID, req.FeatureKey); err != nil {
		log.Printf("feature_interest save error: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
