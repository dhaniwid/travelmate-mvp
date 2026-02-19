package handlers

import (
	"net/http"
	"strconv"

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

// =========================================================
// GAMIFICATION ENDPOINTS (Phase 3)
// =========================================================

// GetLeaderboard handles GET /api/v1/referrals/leaderboard?limit=50
// Returns top referrers ranked by successful referrals
func (h *ReferralHandler) GetLeaderboard(c *gin.Context) {
	// Parse optional limit parameter (default: 50, max: 100)
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	// Fetch leaderboard from service
	leaderboard, err := h.ReferralService.GetLeaderboard(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	// Hide email addresses in public leaderboard
	for i := range leaderboard {
		leaderboard[i].Email = "" // Privacy: don't expose emails publicly
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": leaderboard,
		"count":       len(leaderboard),
	})
}

// GetUserAchievements handles GET /api/v1/user/achievements
// Returns the authenticated user's unlocked achievement badges
func (h *ReferralHandler) GetUserAchievements(c *gin.Context) {
	// Extract user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID missing from context"})
		return
	}

	// Get user's achievements from service
	achievements, err := h.ReferralService.GetUserAchievements(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch achievements"})
		return
	}

	// Get user's rank (if on leaderboard)
	rank, err := h.ReferralService.GetUserRank(c.Request.Context(), userID.(string))

	response := gin.H{
		"achievements":   achievements,
		"total_unlocked": len(achievements),
	}

	if err == nil && rank != nil {
		response["leaderboard_rank"] = rank.Rank
		response["total_referrals"] = rank.TotalReferrals
	}

	c.JSON(http.StatusOK, response)
}

// GetUserRank handles GET /api/v1/referrals/rank
// Returns the authenticated user's current leaderboard rank and referral count.
func (h *ReferralHandler) GetUserRank(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID missing from context"})
		return
	}

	rank, err := h.ReferralService.GetUserRank(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user rank"})
		return
	}

	if rank == nil {
		// User not on leaderboard yet
		c.JSON(http.StatusOK, gin.H{
			"rank":            nil,
			"total_referrals": 0,
			"on_leaderboard":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rank":            rank.Rank,
		"total_referrals": rank.TotalReferrals,
		"on_leaderboard":  true,
	})
}
