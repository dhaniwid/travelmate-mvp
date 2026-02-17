package repositories

import (
	"context"
	"database/sql"
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

// IncrementBonusQuota increases the referrer's bonus trip quota
func (r *ReferralRepository) IncrementBonusQuota(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET bonus_trip_quota = bonus_trip_quota + 1
		WHERE user_id = $1
	`
	_, err := r.DB.ExecContext(ctx, query, userID)
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
