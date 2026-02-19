package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"travelmate/internal/domain"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// GetUserByClerkID fetches a user by their Clerk ID (user_id)
func (r *UserRepository) GetUserByClerkID(ctx context.Context, clerkID string) (*domain.User, error) {
	query := `
		SELECT 
			id, user_id, email, name, 
			subscription_tier, subscription_status, 
			subscription_started_at, subscription_ends_at,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	var subStarted, subEnded sql.NullTime
	var stripeCustID, stripeSubID sql.NullString
	var email, name sql.NullString

	err := r.DB.QueryRowContext(ctx, query, clerkID).Scan(
		&user.ID, &user.UserID, &email, &name,
		&user.SubscriptionTier, &user.SubscriptionStatus,
		&subStarted, &subEnded,
		&stripeCustID, &stripeSubID,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		user.Email = email.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if subStarted.Valid {
		user.SubscriptionStartedAt = &subStarted.Time
	}
	if subEnded.Valid {
		user.SubscriptionEndsAt = &subEnded.Time
	}
	if stripeCustID.Valid {
		user.StripeCustomerID = stripeCustID.String
	}
	if stripeSubID.Valid {
		user.StripeSubscriptionID = stripeSubID.String
	}

	return &user, nil
}

// UpsertUser creates or updates a user based on Clerk ID
func (r *UserRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (user_id, email, name, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
			name = COALESCE(NULLIF(EXCLUDED.name, ''), users.name),
			updated_at = NOW()
		RETURNING id, subscription_tier, subscription_status
	`

	err := r.DB.QueryRowContext(ctx, query,
		user.UserID, user.Email, user.Name,
	).Scan(&user.ID, &user.SubscriptionTier, &user.SubscriptionStatus)

	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	return nil
}

// UpdateUserEmail writes the email (and optionally name) for an existing user.
// Only overwrites if the incoming value is non-empty and the DB column is NULL or empty.
// This is called asynchronously from auth middleware after Clerk profile fetch.
func (r *UserRepository) UpdateUserEmail(ctx context.Context, userID, email, name string) error {
	query := `
		UPDATE users
		SET
			email = CASE WHEN email IS NULL OR email = '' THEN $2 ELSE email END,
			name  = CASE WHEN name  IS NULL OR name  = '' THEN $3 ELSE name  END,
			updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.DB.ExecContext(ctx, query, userID, email, name)
	if err != nil {
		return fmt.Errorf("failed to update user email: %w", err)
	}
	return nil
}

// UpdateSubscription updates the subscription details for a user
func (r *UserRepository) UpdateSubscription(ctx context.Context, userID string, tier, status string, stripeCustID, stripeSubID string) error {
	query := `
		UPDATE users 
		SET 
			subscription_tier = $1,
			subscription_status = $2,
			stripe_customer_id = $3,
			stripe_subscription_id = $4,
			updated_at = NOW()
		WHERE user_id = $5
	`

	_, err := r.DB.ExecContext(ctx, query, tier, status, stripeCustID, stripeSubID, userID)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// GetUserByStripeID fetches a user by their Stripe Customer ID
func (r *UserRepository) GetUserByStripeID(ctx context.Context, stripeCustID string) (*domain.User, error) {
	query := `
		SELECT 
			id, user_id, email, name, 
			subscription_tier, subscription_status, 
			subscription_started_at, subscription_ends_at,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM users
		WHERE stripe_customer_id = $1
	`

	var user domain.User
	var subStarted, subEnded sql.NullTime
	var stripeSubID sql.NullString
	var email, name sql.NullString
	var stripeCustIDDB sql.NullString

	err := r.DB.QueryRowContext(ctx, query, stripeCustID).Scan(
		&user.ID, &user.UserID, &email, &name,
		&user.SubscriptionTier, &user.SubscriptionStatus,
		&subStarted, &subEnded,
		&stripeCustIDDB, &stripeSubID,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by stripe id: %w", err)
	}

	if email.Valid {
		user.Email = email.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if subStarted.Valid {
		user.SubscriptionStartedAt = &subStarted.Time
	}
	if subEnded.Valid {
		user.SubscriptionEndsAt = &subEnded.Time
	}
	if stripeCustIDDB.Valid {
		user.StripeCustomerID = stripeCustIDDB.String
	}
	if stripeSubID.Valid {
		user.StripeSubscriptionID = stripeSubID.String
	}

	return &user, nil
}

// GetUserByEmail fetches a user by their email
func (r *UserRepository) GetUserByEmail(ctx context.Context, emailInput string) (*domain.User, error) {
	query := `
		SELECT 
			id, user_id, email, name, 
			subscription_tier, subscription_status, 
			subscription_started_at, subscription_ends_at,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var subStarted, subEnded sql.NullTime
	var stripeCustID, stripeSubID sql.NullString
	var email, name sql.NullString

	err := r.DB.QueryRowContext(ctx, query, emailInput).Scan(
		&user.ID, &user.UserID, &email, &name,
		&user.SubscriptionTier, &user.SubscriptionStatus,
		&subStarted, &subEnded,
		&stripeCustID, &stripeSubID,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Return nil if user not found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if email.Valid {
		user.Email = email.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if subStarted.Valid {
		user.SubscriptionStartedAt = &subStarted.Time
	}
	if subEnded.Valid {
		user.SubscriptionEndsAt = &subEnded.Time
	}
	if stripeCustID.Valid {
		user.StripeCustomerID = stripeCustID.String
	}
	if stripeSubID.Valid {
		user.StripeSubscriptionID = stripeSubID.String
	}

	return &user, nil
}

// GrantProDays extends or starts a user's PRO subscription by a specific number of days
func (r *UserRepository) GrantProDays(ctx context.Context, userID string, days int) error {
	// 1. Get current subscription state
	query := `
		SELECT subscription_ends_at, subscription_tier
		FROM users
		WHERE user_id = $1
	`
	var endsAt sql.NullTime
	var tier string
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&endsAt, &tier)
	if err != nil {
		return fmt.Errorf("failed to get current subscription: %w", err)
	}

	// 2. Calculate new end date
	newEnd := time.Now().AddDate(0, 0, days)

	// If already PRO and not expired, extend from existing end date
	if tier == "PRO" && endsAt.Valid && endsAt.Time.After(time.Now()) {
		newEnd = endsAt.Time.AddDate(0, 0, days)
	}

	// 3. Update user
	updateQuery := `
		UPDATE users
		SET 
			subscription_tier = 'PRO',
			subscription_status = 'ACTIVE',
			subscription_ends_at = $2,
			updated_at = NOW()
		WHERE user_id = $1
	`
	_, err = r.DB.ExecContext(ctx, updateQuery, userID, newEnd)
	return err
}
