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

// Referral Status Constants
const (
	ReferralStatusPending   = "pending"
	ReferralStatusCompleted = "completed"
)
