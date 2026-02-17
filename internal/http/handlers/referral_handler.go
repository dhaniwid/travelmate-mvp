package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	ReferralService IReferralService
}

func NewReferralHandler(referralService IReferralService) *ReferralHandler {
	return &ReferralHandler{
		ReferralService: referralService,
	}
}

// ClaimReferral handles POST /api/v1/referrals/claim
// Allows a new user to claim a referral code and reward the referrer
func (h *ReferralHandler) ClaimReferral(c *gin.Context) {
	// Extract user ID from auth context (set by AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID missing from context"})
		return
	}

	// Parse request body
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referral code is required"})
		return
	}

	// Process referral
	err := h.ReferralService.ProcessReferral(c.Request.Context(), userID.(string), req.Code)
	if err != nil {
		// Handle specific error cases
		errMsg := err.Error()
		switch errMsg {
		case "invalid referral code":
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid referral code"})
		case "cannot refer yourself":
			c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot use your own referral code"})
		case "user has already been referred":
			c.JSON(http.StatusConflict, gin.H{"error": "You have already claimed a referral code"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process referral"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Referral claimed successfully! The referrer earned +1 trip quota.",
	})
}

// GetReferralInfo handles GET /api/v1/user/referral
// Returns the user's referral code and statistics
func (h *ReferralHandler) GetReferralInfo(c *gin.Context) {
	// Extract user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID missing from context"})
		return
	}

	// Get referral stats
	stats, err := h.ReferralService.GetReferralStats(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referral stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
