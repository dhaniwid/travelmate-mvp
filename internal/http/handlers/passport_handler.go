package handlers

import (
	"context"
	"net/http"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
)

type IPassportService interface {
	GetUserStamps(ctx context.Context, userID string) ([]domain.PassportStamp, error)
	ClaimStamp(ctx context.Context, userID, tripID, city, citySlug string) (*domain.PassportStamp, error)
	CheckStamp(ctx context.Context, userID, citySlug string) ([]domain.PassportStamp, error)
}

type PassportHandler struct {
	Service IPassportService
}

func NewPassportHandler(svc IPassportService) *PassportHandler {
	return &PassportHandler{Service: svc}
}

// GET /api/v1/passport
func (h *PassportHandler) GetUserStamps(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User ID missing"})
		return
	}

	stamps, err := h.Service.GetUserStamps(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "stamps": stamps, "total": len(stamps)})
}

// POST /api/v1/passport/claim
func (h *PassportHandler) ClaimStamp(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User ID missing"})
		return
	}

	var req struct {
		TripID   string `json:"trip_id"`
		City     string `json:"city" binding:"required"`
		CitySlug string `json:"city_slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: err.Error()})
		return
	}

	stamp, err := h.Service.ClaimStamp(c.Request.Context(), userID.(string), req.TripID, req.City, req.CitySlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "claim_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "stamp": stamp})
}

// GET /api/v1/passport/check?city_slug=bali
func (h *PassportHandler) CheckStamp(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: "unauthorized", Message: "User ID missing"})
		return
	}

	citySlug := c.Query("city_slug")
	if citySlug == "" {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: "city_slug query param required"})
		return
	}

	stamps, err := h.Service.CheckStamp(c.Request.Context(), userID.(string), citySlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "stamps": stamps, "has_stamp": len(stamps) > 0})
}
