package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
)

// --- Mock Referral Service ---

type MockReferralService struct {
	ProcessReferralFunc     func(ctx context.Context, newUserID, referralCode string) error
	GetReferralStatsFunc    func(ctx context.Context, userID string) (*domain.ReferralStats, error)
	GetLeaderboardFunc      func(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error)
	GetUserRankFunc         func(ctx context.Context, userID string) (*domain.LeaderboardEntry, error)
	GetUserAchievementsFunc func(ctx context.Context, userID string) ([]domain.Achievement, error)
}

func (m *MockReferralService) ProcessReferral(ctx context.Context, newUserID, referralCode string) error {
	if m.ProcessReferralFunc != nil {
		return m.ProcessReferralFunc(ctx, newUserID, referralCode)
	}
	return nil
}

func (m *MockReferralService) GetReferralStats(ctx context.Context, userID string) (*domain.ReferralStats, error) {
	if m.GetReferralStatsFunc != nil {
		return m.GetReferralStatsFunc(ctx, userID)
	}
	return &domain.ReferralStats{
		ReferralCode:   "MIRU-TEST",
		TotalReferrals: 3,
		BonusQuota:     3,
	}, nil
}

func (m *MockReferralService) GetLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockReferralService) GetUserRank(ctx context.Context, userID string) (*domain.LeaderboardEntry, error) {
	if m.GetUserRankFunc != nil {
		return m.GetUserRankFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockReferralService) GetUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	if m.GetUserAchievementsFunc != nil {
		return m.GetUserAchievementsFunc(ctx, userID)
	}
	return nil, nil
}

// --- Tests ---

// Test 1: Claim Referral - Success (200 OK)
func TestClaimReferral_Success(t *testing.T) {
	mockService := &MockReferralService{
		ProcessReferralFunc: func(ctx context.Context, newUserID, referralCode string) error {
			if newUserID != "user_123" {
				t.Errorf("Expected user_123, got %s", newUserID)
			}
			if referralCode != "MIRU-VALID" {
				t.Errorf("Expected MIRU-VALID, got %s", referralCode)
			}
			return nil
		},
	}

	handler := NewReferralHandler(mockService)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/referrals/claim", func(c *gin.Context) {
		c.Set("userID", "user_123") // Simulate auth middleware
		handler.ClaimReferral(c)
	})

	// Create request
	payload := map[string]string{"code": "MIRU-VALID"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/referrals/claim", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] == "" {
		t.Error("Expected success message in response")
	}
}

// Test 2: Claim Referral - Missing Body (400 Bad Request)
func TestClaimReferral_MissingBody(t *testing.T) {
	mockService := &MockReferralService{}
	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/referrals/claim", func(c *gin.Context) {
		c.Set("userID", "user_123")
		handler.ClaimReferral(c)
	})

	// Empty body
	req := httptest.NewRequest("POST", "/referrals/claim", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test 3: Claim Referral - Invalid Code (404 Not Found)
func TestClaimReferral_InvalidCode(t *testing.T) {
	mockService := &MockReferralService{
		ProcessReferralFunc: func(ctx context.Context, newUserID, referralCode string) error {
			return &ReferralError{Type: "invalid_code", Message: "invalid referral code"}
		},
	}

	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/referrals/claim", func(c *gin.Context) {
		c.Set("userID", "user_123")
		handler.ClaimReferral(c)
	})

	payload := map[string]string{"code": "INVALID"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/referrals/claim", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test 4: Claim Referral - Self Referral (400 Bad Request)
func TestClaimReferral_SelfReferral(t *testing.T) {
	mockService := &MockReferralService{
		ProcessReferralFunc: func(ctx context.Context, newUserID, referralCode string) error {
			return &ReferralError{Type: "self_referral", Message: "cannot refer yourself"}
		},
	}

	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/referrals/claim", func(c *gin.Context) {
		c.Set("userID", "user_123")
		handler.ClaimReferral(c)
	})

	payload := map[string]string{"code": "MIRU-SELF"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/referrals/claim", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test 5: Claim Referral - Already Claimed (409 Conflict)
func TestClaimReferral_AlreadyClaimed(t *testing.T) {
	mockService := &MockReferralService{
		ProcessReferralFunc: func(ctx context.Context, newUserID, referralCode string) error {
			return &ReferralError{Type: "duplicate", Message: "user has already been referred"}
		},
	}

	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/referrals/claim", func(c *gin.Context) {
		c.Set("userID", "user_123")
		handler.ClaimReferral(c)
	})

	payload := map[string]string{"code": "MIRU-DUP"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/referrals/claim", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// Test 6: Get Referral Info - Success (200 OK)
func TestGetReferralInfo_Success(t *testing.T) {
	mockService := &MockReferralService{
		GetReferralStatsFunc: func(ctx context.Context, userID string) (*domain.ReferralStats, error) {
			return &domain.ReferralStats{
				ReferralCode:   "MIRU-ABC1",
				TotalReferrals: 5,
				BonusQuota:     5,
			}, nil
		},
	}

	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/referral", func(c *gin.Context) {
		c.Set("userID", "user_123")
		handler.GetReferralInfo(c)
	})

	req := httptest.NewRequest("GET", "/user/referral", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats domain.ReferralStats
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.ReferralCode != "MIRU-ABC1" {
		t.Errorf("Expected MIRU-ABC1, got %s", stats.ReferralCode)
	}

	if stats.TotalReferrals != 5 {
		t.Errorf("Expected 5 referrals, got %d", stats.TotalReferrals)
	}

	if stats.BonusQuota != 5 {
		t.Errorf("Expected 5 bonus quota, got %d", stats.BonusQuota)
	}
}

// Test 7: Get Referral Info - Unauthorized (401)
func TestGetReferralInfo_Unauthorized(t *testing.T) {
	mockService := &MockReferralService{}
	handler := NewReferralHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/referral", handler.GetReferralInfo) // No auth middleware

	req := httptest.NewRequest("GET", "/user/referral", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// Helper type for structured error handling
type ReferralError struct {
	Type    string
	Message string
}

func (e *ReferralError) Error() string {
	return e.Message
}
