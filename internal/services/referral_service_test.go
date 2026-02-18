package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"travelmate/internal/domain"
)

// --- Mock Referral Repository ---

type MockReferralRepo struct {
	CreateReferralFunc        func(ctx context.Context, referral *domain.Referral) error
	GetUserByReferralCodeFunc func(ctx context.Context, code string) (*domain.User, error)
	IncrementBonusQuotaFunc   func(ctx context.Context, userID string, amount int) error
	CheckReferralExistsFunc   func(ctx context.Context, referredUserID string) (bool, error)
	GetReferralCountFunc      func(ctx context.Context, referrerID string) (int, error)
	GetLeaderboardFunc        func(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error)
	GetUserRankFunc           func(ctx context.Context, userID string) (*domain.LeaderboardEntry, error)
	GetUserAchievementsFunc   func(ctx context.Context, userID string) ([]domain.Achievement, error)
	UnlockAchievementFunc     func(ctx context.Context, userID string, achievement domain.Achievement) error
	RefreshLeaderboardFunc    func(ctx context.Context) error
}

func (m *MockReferralRepo) CreateReferral(ctx context.Context, referral *domain.Referral) error {
	if m.CreateReferralFunc != nil {
		return m.CreateReferralFunc(ctx, referral)
	}
	return nil
}

func (m *MockReferralRepo) GetReferralsByReferrer(ctx context.Context, referrerID string) ([]domain.Referral, error) {
	return nil, nil
}

func (m *MockReferralRepo) GetUserByReferralCode(ctx context.Context, code string) (*domain.User, error) {
	if m.GetUserByReferralCodeFunc != nil {
		return m.GetUserByReferralCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockReferralRepo) IncrementBonusQuota(ctx context.Context, userID string, amount int) error {
	if m.IncrementBonusQuotaFunc != nil {
		return m.IncrementBonusQuotaFunc(ctx, userID, amount)
	}
	return nil
}

func (m *MockReferralRepo) CheckReferralExists(ctx context.Context, referredUserID string) (bool, error) {
	if m.CheckReferralExistsFunc != nil {
		return m.CheckReferralExistsFunc(ctx, referredUserID)
	}
	return false, nil
}

func (m *MockReferralRepo) GetReferralCount(ctx context.Context, referrerID string) (int, error) {
	if m.GetReferralCountFunc != nil {
		return m.GetReferralCountFunc(ctx, referrerID)
	}
	return 0, nil
}

func (m *MockReferralRepo) GetLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockReferralRepo) GetUserRank(ctx context.Context, userID string) (*domain.LeaderboardEntry, error) {
	if m.GetUserRankFunc != nil {
		return m.GetUserRankFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockReferralRepo) GetUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	if m.GetUserAchievementsFunc != nil {
		return m.GetUserAchievementsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockReferralRepo) UnlockAchievement(ctx context.Context, userID string, achievement domain.Achievement) error {
	if m.UnlockAchievementFunc != nil {
		return m.UnlockAchievementFunc(ctx, userID, achievement)
	}
	return nil
}

func (m *MockReferralRepo) RefreshLeaderboard(ctx context.Context) error {
	if m.RefreshLeaderboardFunc != nil {
		return m.RefreshLeaderboardFunc(ctx)
	}
	return nil
}

// --- Tests ---

// Test 1: Code Generation Format and Consistency
func TestGenerateReferralCode(t *testing.T) {
	// Run 1000 times to check for consistency
	codes := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		code, err := GenerateReferralCode()

		// Check no error
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Check format: MIRU-XXXX
		if !strings.HasPrefix(code, "MIRU-") {
			t.Errorf("Code should start with 'MIRU-', got %s", code)
		}

		// Check length (MIRU- = 5 chars + 6 chars = 11 total)
		if len(code) != 11 {
			t.Errorf("Code should be 11 characters, got %d: %s", len(code), code)
		}

		// Check for valid characters (no confusing chars)
		suffix := code[5:] // Get XXXXXX part
		for _, char := range suffix {
			if !isValidReferralChar(char) {
				t.Errorf("Code contains invalid character: %c in %s", char, code)
			}
		}

		// Track for collision detection
		if codes[code] {
			t.Errorf("Collision detected! Code %s generated twice", code)
		}
		codes[code] = true
	}

	// Verify we generated 1000 unique codes
	if len(codes) != 1000 {
		t.Errorf("Expected 1000 unique codes, got %d", len(codes))
	}
}

// Helper to validate referral code characters
func isValidReferralChar(c rune) bool {
	validChars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	return strings.ContainsRune(validChars, c)
}

// Test 2: Successful Referral Processing
func TestProcessReferral_Success(t *testing.T) {
	referralCreated := false
	bonusIncremented := false

	mockReferralRepo := &MockReferralRepo{
		GetUserByReferralCodeFunc: func(ctx context.Context, code string) (*domain.User, error) {
			if code == "MIRU-TEST" {
				return &domain.User{
					UserID:       "referrer_123",
					ReferralCode: "MIRU-TEST",
				}, nil
			}
			return nil, nil
		},
		CheckReferralExistsFunc: func(ctx context.Context, referredUserID string) (bool, error) {
			return false, nil // User hasn't been referred yet
		},
		CreateReferralFunc: func(ctx context.Context, referral *domain.Referral) error {
			if referral.ReferrerID != "referrer_123" {
				t.Errorf("Expected referrer_123, got %s", referral.ReferrerID)
			}
			if referral.ReferredUserID != "new_user_456" {
				t.Errorf("Expected new_user_456, got %s", referral.ReferredUserID)
			}
			if referral.Status != domain.ReferralStatusCompleted {
				t.Errorf("Expected completed status, got %s", referral.Status)
			}
			referralCreated = true
			return nil
		},
		IncrementBonusQuotaFunc: func(ctx context.Context, userID string, amount int) error {
			if userID != "referrer_123" {
				t.Errorf("Expected referrer_123, got %s", userID)
			}
			if amount != 1 {
				t.Errorf("Expected amount 1, got %d", amount)
			}
			bonusIncremented = true
			return nil
		},
	}

	mockUserRepo := &MockUserRepo{}
	service := NewReferralService(mockReferralRepo, mockUserRepo)

	err := service.ProcessReferral(context.Background(), "new_user_456", "MIRU-TEST")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !referralCreated {
		t.Error("Referral record was not created")
	}

	if !bonusIncremented {
		t.Error("Bonus quota was not incremented")
	}
}

// Test 3: Self-Referral Prevention
func TestProcessReferral_Fail_SelfReferral(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{
		GetUserByReferralCodeFunc: func(ctx context.Context, code string) (*domain.User, error) {
			return &domain.User{
				UserID:       "user_123",
				ReferralCode: "MIRU-SELF",
			}, nil
		},
	}

	mockUserRepo := &MockUserRepo{}
	service := NewReferralService(mockReferralRepo, mockUserRepo)

	err := service.ProcessReferral(context.Background(), "user_123", "MIRU-SELF")

	if err == nil {
		t.Fatal("Expected error for self-referral, got nil")
	}

	if !strings.Contains(err.Error(), "cannot refer yourself") {
		t.Errorf("Expected 'cannot refer yourself' error, got %v", err)
	}
}

// Test 4: Duplicate Referral Prevention
func TestProcessReferral_Fail_AlreadyReferred(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{
		GetUserByReferralCodeFunc: func(ctx context.Context, code string) (*domain.User, error) {
			return &domain.User{
				UserID:       "referrer_123",
				ReferralCode: "MIRU-DUP",
			}, nil
		},
		CheckReferralExistsFunc: func(ctx context.Context, referredUserID string) (bool, error) {
			return true, nil // User already referred
		},
	}

	mockUserRepo := &MockUserRepo{}
	service := NewReferralService(mockReferralRepo, mockUserRepo)

	err := service.ProcessReferral(context.Background(), "user_456", "MIRU-DUP")

	if err == nil {
		t.Fatal("Expected error for duplicate referral, got nil")
	}

	if !strings.Contains(err.Error(), "already been referred") {
		t.Errorf("Expected 'already been referred' error, got %v", err)
	}
}

// Test 5: Invalid Referral Code
func TestProcessReferral_Fail_InvalidCode(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{
		GetUserByReferralCodeFunc: func(ctx context.Context, code string) (*domain.User, error) {
			return nil, nil // Code not found
		},
	}

	mockUserRepo := &MockUserRepo{}
	service := NewReferralService(mockReferralRepo, mockUserRepo)

	err := service.ProcessReferral(context.Background(), "user_789", "INVALID-CODE")

	if err == nil {
		t.Fatal("Expected error for invalid code, got nil")
	}

	if !strings.Contains(err.Error(), "invalid referral code") {
		t.Errorf("Expected 'invalid referral code' error, got %v", err)
	}
}

// Test 6: Get Referral Stats
func TestGetReferralStats_Success(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{
		GetReferralCountFunc: func(ctx context.Context, referrerID string) (int, error) {
			if referrerID == "user_123" {
				return 5, nil
			}
			return 0, nil
		},
	}

	mockUserRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "user_123" {
				return &domain.User{
					UserID:         "user_123",
					ReferralCode:   "MIRU-STAT",
					BonusTripQuota: 5,
				}, nil
			}
			return nil, nil
		},
	}

	service := NewReferralService(mockReferralRepo, mockUserRepo)

	stats, err := service.GetReferralStats(context.Background(), "user_123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.ReferralCode != "MIRU-STAT" {
		t.Errorf("Expected MIRU-STAT, got %s", stats.ReferralCode)
	}

	if stats.TotalReferrals != 5 {
		t.Errorf("Expected 5 referrals, got %d", stats.TotalReferrals)
	}

	if stats.BonusQuota != 5 {
		t.Errorf("Expected 5 bonus quota, got %d", stats.BonusQuota)
	}
}

// Test 7: Get Referral Stats - User Not Found
func TestGetReferralStats_UserNotFound(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{}

	mockUserRepo := &MockUserRepo{
		GetUserByClerkIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, nil // User not found
		},
	}

	service := NewReferralService(mockReferralRepo, mockUserRepo)

	_, err := service.GetReferralStats(context.Background(), "nonexistent_user")

	if err == nil {
		t.Fatal("Expected error for nonexistent user, got nil")
	}

	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("Expected 'user not found' error, got %v", err)
	}
}

// Test 8: Database Error Handling
func TestProcessReferral_DatabaseError(t *testing.T) {
	mockReferralRepo := &MockReferralRepo{
		GetUserByReferralCodeFunc: func(ctx context.Context, code string) (*domain.User, error) {
			return &domain.User{UserID: "referrer_123", ReferralCode: "MIRU-ERR"}, nil
		},
		CheckReferralExistsFunc: func(ctx context.Context, referredUserID string) (bool, error) {
			return false, nil
		},
		CreateReferralFunc: func(ctx context.Context, referral *domain.Referral) error {
			return errors.New("database connection failed")
		},
	}

	mockUserRepo := &MockUserRepo{}
	service := NewReferralService(mockReferralRepo, mockUserRepo)

	err := service.ProcessReferral(context.Background(), "user_456", "MIRU-ERR")

	if err == nil {
		t.Fatal("Expected database error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create referral") {
		t.Errorf("Expected 'failed to create referral' error, got %v", err)
	}
}
