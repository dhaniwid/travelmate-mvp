package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"
	"travelmate/internal/domain"
)

// ReferralRepo defines the interface for referral data access
type ReferralRepo interface {
	CreateReferral(ctx context.Context, referral *domain.Referral) error
	GetReferralsByReferrer(ctx context.Context, referrerID string) ([]domain.Referral, error)
	GetUserByReferralCode(ctx context.Context, code string) (*domain.User, error)
	IncrementBonusQuota(ctx context.Context, userID string, amount int) error
	CheckReferralExists(ctx context.Context, referredUserID string) (bool, error)
	GetReferralCount(ctx context.Context, referrerID string) (int, error)

	// Phase 3: Gamification
	GetLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error)
	GetUserRank(ctx context.Context, userID string) (*domain.LeaderboardEntry, error)
	GetUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error)
	UnlockAchievement(ctx context.Context, userID string, achievement domain.Achievement) error
	RefreshLeaderboard(ctx context.Context) error
}

type ReferralService struct {
	ReferralRepo ReferralRepo
	UserRepo     UserRepo
}

func NewReferralService(referralRepo ReferralRepo, userRepo UserRepo) *ReferralService {
	return &ReferralService{
		ReferralRepo: referralRepo,
		UserRepo:     userRepo,
	}
}

// GenerateReferralCode creates a unique referral code in format: MIRU-XXXX
// Uses crypto/rand for secure randomness
func GenerateReferralCode() (string, error) {
	// Use characters that are easy to read (exclude confusing chars like 0, O, I, 1)
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const codeLength = 6

	code := "MIRU-"
	for i := 0; i < codeLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}
		code += string(chars[num.Int64()])
	}

	return code, nil
}

// ProcessReferral handles the referral claim process
// 1. Validates the referral code exists
// 2. Checks for self-referral
// 3. Ensures user hasn't been referred before
// 4. Creates referral record
// 5. Rewards the referrer with +1 bonus quota
func (s *ReferralService) ProcessReferral(ctx context.Context, newUserID, referralCode string) error {
	// 1. Validate referral code exists
	referrer, err := s.ReferralRepo.GetUserByReferralCode(ctx, referralCode)
	if err != nil {
		return fmt.Errorf("failed to validate referral code: %w", err)
	}
	if referrer == nil {
		return fmt.Errorf("invalid referral code")
	}

	// 2. Check for self-referral
	if referrer.UserID == newUserID {
		return fmt.Errorf("cannot refer yourself")
	}

	// 3. Check if user has already been referred
	alreadyReferred, err := s.ReferralRepo.CheckReferralExists(ctx, newUserID)
	if err != nil {
		return fmt.Errorf("failed to check referral status: %w", err)
	}
	if alreadyReferred {
		return fmt.Errorf("user has already been referred")
	}

	// 4. Create referral record
	referral := &domain.Referral{
		ReferrerID:     referrer.UserID,
		ReferredUserID: newUserID,
		Status:         domain.ReferralStatusCompleted,
	}
	if err := s.ReferralRepo.CreateReferral(ctx, referral); err != nil {
		return fmt.Errorf("failed to create referral: %w", err)
	}

	// 5. Reward referrer with +1 bonus quota
	if err := s.ReferralRepo.IncrementBonusQuota(ctx, referrer.UserID, 1); err != nil {
		return fmt.Errorf("failed to increment bonus quota: %w", err)
	}

	// 6. Check for milestone achievements (Phase 3)
	// We run this as a best-effort, logging any errors but not failing the referral
	go func() {
		// Use background context with timeout for background task
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.CheckAndUnlockMilestones(bgCtx, referrer.UserID); err != nil {
			log.Printf("⚠️ Failed to check milestones for user %s: %v", referrer.UserID, err)
		}
	}()

	return nil
}

// GetReferralStats returns the user's referral statistics
func (s *ReferralService) GetReferralStats(ctx context.Context, userID string) (*domain.ReferralStats, error) {
	// Get user to fetch referral code and bonus quota
	user, err := s.UserRepo.GetUserByClerkID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get total referral count
	count, err := s.ReferralRepo.GetReferralCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get referral count: %w", err)
	}

	return &domain.ReferralStats{
		ReferralCode:   user.ReferralCode,
		TotalReferrals: count,
		BonusQuota:     user.BonusTripQuota,
	}, nil
}

// =========================================================
// GAMIFICATION METHODS (Phase 3)
// =========================================================

// GetMilestoneDefinitions returns the reward structure for all tiers
func GetMilestoneDefinitions() []domain.MilestoneReward {
	return []domain.MilestoneReward{
		{Tier: domain.TierBronze, ReferralsNeeded: 1, BadgeName: "First Blood", BonusTrips: 1, ProDays: 0},
		{Tier: domain.TierSilver, ReferralsNeeded: 5, BadgeName: "Rising Star", BonusTrips: 3, ProDays: 0},
		{Tier: domain.TierGold, ReferralsNeeded: 10, BadgeName: "Growth Master", BonusTrips: 5, ProDays: 7},
		{Tier: domain.TierPlatinum, ReferralsNeeded: 25, BadgeName: "Elite Recruiter", BonusTrips: 10, ProDays: 30},
		{Tier: domain.TierDiamond, ReferralsNeeded: 50, BadgeName: "Legendary Ambassador", BonusTrips: 20, ProDays: 90},
	}
}

// CheckAndUnlockMilestones evaluates user's referrals and unlocks new achievements
// This should be called after ProcessReferral completes successfully
func (s *ReferralService) CheckAndUnlockMilestones(ctx context.Context, userID string) error {
	// Get current referral count
	count, err := s.ReferralRepo.GetReferralCount(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get referral count: %w", err)
	}

	// Check each milestone tier
	milestones := GetMilestoneDefinitions()
	for _, milestone := range milestones {
		if count == milestone.ReferralsNeeded {
			// User just hit this milestone!
			achievement := domain.Achievement{
				ID:          fmt.Sprintf("%s_%s", userID, milestone.Tier),
				Name:        milestone.BadgeName,
				Description: fmt.Sprintf("Earned %d successful referrals", milestone.ReferralsNeeded),
				Icon:        getTierIcon(milestone.Tier),
				Tier:        milestone.Tier,
				UnlockedAt:  time.Now(),
			}

			// 1. Unlock achievement in database
			if err := s.ReferralRepo.UnlockAchievement(ctx, userID, achievement); err != nil {
				log.Printf("⚠️ Failed to unlock achievement %s for user %s: %v", achievement.Name, userID, err)
			}

			// 2. Apply rewards (bonus trips, PRO days)
			if milestone.BonusTrips > 0 {
				if err := s.ReferralRepo.IncrementBonusQuota(ctx, userID, milestone.BonusTrips); err != nil {
					log.Printf("⚠️ Failed to apply bonus trips reward for user %s: %v", userID, err)
				}
			}
			if milestone.ProDays > 0 {
				if err := s.UserRepo.GrantProDays(ctx, userID, milestone.ProDays); err != nil {
					log.Printf("⚠️ Failed to grant PRO days reward for user %s: %v", userID, err)
				}
			}

			// 3. Refresh leaderboard to reflect new achievement
			if err := s.ReferralRepo.RefreshLeaderboard(ctx); err != nil {
				log.Printf("⚠️ Failed to refresh leaderboard: %v", err)
			}

			fmt.Printf("🎉 Milestone unlocked for user %s: %s (%s)\n", userID, achievement.Name, milestone.Tier)
		}
	}

	return nil
}

// getTierIcon returns the emoji or icon name for a tier
func getTierIcon(tier string) string {
	icons := map[string]string{
		domain.TierBronze:   "🥉",
		domain.TierSilver:   "🥈",
		domain.TierGold:     "🥇",
		domain.TierPlatinum: "💎",
		domain.TierDiamond:  "👑",
	}
	return icons[tier]
}

// GetLeaderboard fetches top referrers from the leaderboard
func (s *ReferralService) GetLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	return s.ReferralRepo.GetLeaderboard(ctx, limit)
}

// GetUserRank fetches a specific user's leaderboard position
func (s *ReferralService) GetUserRank(ctx context.Context, userID string) (*domain.LeaderboardEntry, error) {
	return s.ReferralRepo.GetUserRank(ctx, userID)
}

// GetUserAchievements fetches all unlocked achievements for a user
func (s *ReferralService) GetUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	return s.ReferralRepo.GetUserAchievements(ctx, userID)
}
