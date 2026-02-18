package domain

import "time"

// ==========================================
// REFERRAL SYSTEM MODELS
// ==========================================

// Referral represents a referral relationship between users
type Referral struct {
	ID             string    `json:"id" db:"id"`
	ReferrerID     string    `json:"referrer_id" db:"referrer_id"`
	ReferredUserID string    `json:"referred_user_id" db:"referred_user_id"`
	Status         string    `json:"status" db:"status"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ReferralStats contains user's referral statistics
type ReferralStats struct {
	ReferralCode   string `json:"referral_code"`
	TotalReferrals int    `json:"total_referrals"`
	BonusQuota     int    `json:"bonus_quota"`
}

// ==========================================
// GAMIFICATION MODELS (Phase 3)
// ==========================================

// Achievement represents an unlocked referral milestone badge
type Achievement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"` // Lucide icon name or emoji
	Tier        string    `json:"tier"` // bronze, silver, gold, platinum, diamond
	UnlockedAt  time.Time `json:"unlocked_at"`
}

// LeaderboardEntry represents a user's position on the referral leaderboard
type LeaderboardEntry struct {
	UserID           string        `json:"user_id" db:"user_id"`
	Name             string        `json:"name" db:"name"`
	Email            string        `json:"email" db:"email"` // Hidden from public view
	ReferralCode     string        `json:"referral_code" db:"referral_code"`
	Rank             int           `json:"rank" db:"rank"`
	TotalReferrals   int           `json:"total_referrals" db:"total_referrals"`
	BonusQuota       int           `json:"bonus_quota" db:"bonus_trip_quota"`
	Achievements     []Achievement `json:"achievements"`
	FirstReferralAt  time.Time     `json:"first_referral_at" db:"first_referral_at"`
	LatestReferralAt time.Time     `json:"latest_referral_at" db:"latest_referral_at"`
}

// MilestoneReward defines the reward structure for each achievement tier
type MilestoneReward struct {
	Tier            string `json:"tier"`
	ReferralsNeeded int    `json:"referrals_needed"`
	BadgeName       string `json:"badge_name"`
	BonusTrips      int    `json:"bonus_trips"`
	ProDays         int    `json:"pro_days"` // 0 = no PRO upgrade
}

// Milestone tier constants
const (
	TierBronze   = "bronze"
	TierSilver   = "silver"
	TierGold     = "gold"
	TierPlatinum = "platinum"
	TierDiamond  = "diamond"
)

// Referral Status Constants
const (
	ReferralStatusPending   = "pending"
	ReferralStatusCompleted = "completed"
)
