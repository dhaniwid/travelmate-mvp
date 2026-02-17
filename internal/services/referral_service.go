package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"travelmate/internal/domain"
)

// ReferralRepo defines the interface for referral data access
type ReferralRepo interface {
	CreateReferral(ctx context.Context, referral *domain.Referral) error
	GetReferralsByReferrer(ctx context.Context, referrerID string) ([]domain.Referral, error)
	GetUserByReferralCode(ctx context.Context, code string) (*domain.User, error)
	IncrementBonusQuota(ctx context.Context, userID string) error
	CheckReferralExists(ctx context.Context, referredUserID string) (bool, error)
	GetReferralCount(ctx context.Context, referrerID string) (int, error)
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
	const codeLength = 4

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
	if err := s.ReferralRepo.IncrementBonusQuota(ctx, referrer.UserID); err != nil {
		return fmt.Errorf("failed to increment bonus quota: %w", err)
	}

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
