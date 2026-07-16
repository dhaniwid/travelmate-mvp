package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"travelmate/internal/domain"
)

type ReferralRepository struct {
	DB *sql.DB
}

func NewReferralRepository(db *sql.DB) *ReferralRepository {
	return &ReferralRepository{DB: db}
}

// CreateReferral records a new referral relationship
func (r *ReferralRepository) CreateReferral(ctx context.Context, referral *domain.Referral) error {
	query := `
		INSERT INTO referrals (referrer_id, referred_user_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.DB.QueryRowContext(
		ctx,
		query,
		referral.ReferrerID,
		referral.ReferredUserID,
		referral.Status,
	).Scan(&referral.ID, &referral.CreatedAt)
}

// GetReferralsByReferrer fetches all referrals made by a user
func (r *ReferralRepository) GetReferralsByReferrer(ctx context.Context, referrerID string) ([]domain.Referral, error) {
	query := `
		SELECT id, referrer_id, referred_user_id, status, created_at
		FROM referrals
		WHERE referrer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, referrerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []domain.Referral
	for rows.Next() {
		var ref domain.Referral
		if err := rows.Scan(&ref.ID, &ref.ReferrerID, &ref.ReferredUserID, &ref.Status, &ref.CreatedAt); err != nil {
			return nil, err
		}
		referrals = append(referrals, ref)
	}
	return referrals, nil
}

// GetUserByReferralCode finds a user by their referral code
func (r *ReferralRepository) GetUserByReferralCode(ctx context.Context, code string) (*domain.User, error) {
	query := `
		SELECT id, user_id, email, name, subscription_tier, subscription_status,
		       referral_code, bonus_trip_quota, created_at, updated_at
		FROM users
		WHERE referral_code = $1
	`
	var user domain.User
	err := r.DB.QueryRowContext(ctx, query, code).Scan(
		&user.ID, &user.UserID, &user.Email, &user.Name, &user.SubscriptionTier, &user.SubscriptionStatus,
		&user.ReferralCode, &user.BonusTripQuota, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

// IncrementBonusQuota increases the referrer's bonus trip quota by a specific amount
func (r *ReferralRepository) IncrementBonusQuota(ctx context.Context, userID string, amount int) error {
	query := `
		UPDATE users
		SET bonus_trip_quota = bonus_trip_quota + $2
		WHERE user_id = $1
	`
	_, err := r.DB.ExecContext(ctx, query, userID, amount)
	return err
}

// CheckReferralExists checks if a user has already been referred
func (r *ReferralRepository) CheckReferralExists(ctx context.Context, referredUserID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM referrals WHERE referred_user_id = $1
		)
	`
	var exists bool
	err := r.DB.QueryRowContext(ctx, query, referredUserID).Scan(&exists)
	return exists, err
}

// GetReferralCount returns the total number of successful referrals for a user
func (r *ReferralRepository) GetReferralCount(ctx context.Context, referrerID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM referrals
		WHERE referrer_id = $1 AND status = 'completed'
	`
	var count int
	err := r.DB.QueryRowContext(ctx, query, referrerID).Scan(&count)
	return count, err
}

// =========================================================
// GAMIFICATION METHODS (Phase 3)
// =========================================================

// GetLeaderboard fetches top referrers from materialized view
func (r *ReferralRepository) GetLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 100 // Cap at 100 for performance
	}

	query := `
		SELECT 
			user_id, name, email, referral_code, rank,
			total_referrals, bonus_trip_quota as bonus_quota,
			first_referral_at, latest_referral_at
		FROM referral_leaderboard
		WHERE rank <= $1
		ORDER BY rank ASC
	`
	rows, err := r.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	for rows.Next() {
		var entry domain.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.Name, &entry.Email, &entry.ReferralCode, &entry.Rank,
			&entry.TotalReferrals, &entry.BonusQuota,
			&entry.FirstReferralAt, &entry.LatestReferralAt,
		); err != nil {
			return nil, err
		}
		// Achievements will be populated by service layer
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetUserRank fetches a specific user's leaderboard rank
func (r *ReferralRepository) GetUserRank(ctx context.Context, userID string) (*domain.LeaderboardEntry, error) {
	query := `
		SELECT 
			user_id, name, email, referral_code, rank,
			total_referrals, bonus_trip_quota as bonus_quota,
			first_referral_at, latest_referral_at
		FROM referral_leaderboard
		WHERE user_id = $1
	`
	var entry domain.LeaderboardEntry
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(
		&entry.UserID, &entry.Name, &entry.Email, &entry.ReferralCode, &entry.Rank,
		&entry.TotalReferrals, &entry.BonusQuota,
		&entry.FirstReferralAt, &entry.LatestReferralAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // User not on leaderboard (0 referrals)
	}
	return &entry, err
}

// GetUserAchievements fetches achievements from user's JSONB field
func (r *ReferralRepository) GetUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	query := `
		SELECT achievements_unlocked
		FROM users
		WHERE user_id = $1
	`
	var achievementsJSON []byte
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&achievementsJSON)
	if err != nil {
		return nil, err
	}

	var achievements []domain.Achievement
	if len(achievementsJSON) > 0 && string(achievementsJSON) != "[]" {
		if err := json.Unmarshal(achievementsJSON, &achievements); err != nil {
			return nil, err
		}
	}
	return achievements, nil
}

// UnlockAchievement adds a new achievement to user's unlocked list
func (r *ReferralRepository) UnlockAchievement(ctx context.Context, userID string, achievement domain.Achievement) error {
	query := `
		UPDATE users
		SET achievements_unlocked = achievements_unlocked || $1::jsonb
		WHERE user_id = $2
	`
	achievementJSON := `{
		"id": "` + achievement.ID + `",
		"name": "` + achievement.Name + `",
		"description": "` + achievement.Description + `",
		"icon": "` + achievement.Icon + `",
		"tier": "` + achievement.Tier + `",
		"unlocked_at": "` + achievement.UnlockedAt.Format("2006-01-02T15:04:05Z07:00") + `"
	}`
	_, err := r.DB.ExecContext(ctx, query, achievementJSON, userID)
	return err
}

// RefreshLeaderboard manually refreshes the materialized view
func (r *ReferralRepository) RefreshLeaderboard(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, "SELECT refresh_referral_leaderboard()")
	return err
}
