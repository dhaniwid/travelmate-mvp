package services

import (
	"context"
	"fmt"
	"log"
	"time"
	"travelmate/internal/domain"
)

// UserRepo defines user data access methods
type UserRepo interface {
	GetUserByClerkID(ctx context.Context, clerkID string) (*domain.User, error)
	UpsertUser(ctx context.Context, user *domain.User) error
	UpdateSubscription(ctx context.Context, userID, tier, status, stripeCustID, stripeSubID string) error
	GrantProDays(ctx context.Context, userID string, days int) error
}

// SubRepo defines subscription data access methods
type SubRepo interface {
	GetQuota(ctx context.Context, userID, month string) (*domain.TripQuota, error)
	IncrementQuota(ctx context.Context, userID, month string) error
	LogSubscriptionEvent(ctx context.Context, event *domain.SubscriptionEvent) error
}

// PaymentGateway defines payment processing methods
type PaymentGateway interface {
	CreateCheckoutSession(userID, email, priceID string) (string, error)
}

type SubscriptionService struct {
	UserRepo     UserRepo
	SubRepo      SubRepo
	StripeClient PaymentGateway
}

func NewSubscriptionService(
	userRepo UserRepo,
	subRepo SubRepo,
	stripeClient PaymentGateway,
) *SubscriptionService {
	return &SubscriptionService{
		UserRepo:     userRepo,
		SubRepo:      subRepo,
		StripeClient: stripeClient,
	}
}

// CreateCheckoutSession initiates a subscription update
func (s *SubscriptionService) CreateCheckoutSession(userID, email, priceID string) (string, error) {
	return s.StripeClient.CreateCheckoutSession(userID, email, priceID)
}

// GetUserSubscription returns subscription details for a user.
// It also ensures the user exists in our local DB (lazy creation).
func (s *SubscriptionService) GetUserSubscription(ctx context.Context, userID, email, name string) (*domain.User, error) {
	// 1. Try to get user
	user, err := s.UserRepo.GetUserByClerkID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. If user doesn't exist, create them
	if user == nil {
		newUser := &domain.User{
			UserID:             userID,
			Email:              email,
			Name:               name,
			SubscriptionTier:   "FREE",
			SubscriptionStatus: "ACTIVE",
		}
		if err := s.UserRepo.UpsertUser(ctx, newUser); err != nil {
			return nil, err
		}
		return newUser, nil
	}

	// 3. LAZY EXPIRATION CHECK (M-127)
	// If user is PRO but the subscription period has ended, downgrade them on the fly.
	if user.SubscriptionTier == "PRO" && user.SubscriptionEndsAt != nil {
		if time.Now().After(*user.SubscriptionEndsAt) {
			log.Printf("Lazy Expiration: User %s has expired subscription. Downgrading to FREE.", userID)

			// Update local object immediately for the response
			user.SubscriptionTier = "FREE"
			user.SubscriptionStatus = "EXPIRED"

			// Fire-and-forget DB update to sync the state
			go func(uid string) {
				// We use a background context or a timeout-safe context for the background update
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.UserRepo.UpdateSubscription(bgCtx, uid, "FREE", "EXPIRED", user.StripeCustomerID, user.StripeSubscriptionID); err != nil {
					log.Printf("Failed to async-update expired subscription for %s: %v", uid, err)
				}
			}(userID)
		}
	}

	return user, nil
}

// GetUserQuota returns standard quota info
func (s *SubscriptionService) GetUserQuota(ctx context.Context, userID, email string) (*domain.TripQuota, error) {
	// First ensure user exists (in case they call quota endpoint directly)
	user, err := s.GetUserSubscription(ctx, userID, email, "")
	if err != nil {
		return nil, err
	}

	// If PRO, return virtual unlimited quota
	if user.SubscriptionTier == "PRO" && user.SubscriptionStatus == "ACTIVE" {
		return &domain.TripQuota{
			UserID:       userID,
			Month:        time.Now().Format("2006-01"),
			TripsCreated: 0,
			QuotaLimit:   0,
			Remaining:    9999,
			IsUnlimited:  true,
		}, nil
	}

	// If FREE, fetch from DB
	month := time.Now().Format("2006-01")
	quota, err := s.SubRepo.GetQuota(ctx, userID, month)
	if err != nil {
		return nil, err
	}

	// 🎁 REFERRAL BONUS: Add bonus quota from referrals
	effectiveLimit := quota.QuotaLimit + user.BonusTripQuota

	// Calculate remaining
	quota.Remaining = effectiveLimit - quota.TripsCreated
	if quota.Remaining < 0 {
		quota.Remaining = 0
	}
	quota.IsUnlimited = false

	return quota, nil
}

// IsPROUser returns true if the user has an active PRO subscription.
// Also auto-creates the user in DB if they don't exist (lazy sync).
func (s *SubscriptionService) IsPROUser(ctx context.Context, userID string) (bool, error) {
	user, err := s.UserRepo.GetUserByClerkID(ctx, userID)
	if err != nil {
		return false, err
	}

	if user == nil {
		log.Printf("🧬 [LAZY SYNC] User %s not found in DB. Auto-creating as FREE.", userID)
		user = &domain.User{
			UserID:             userID,
			Email:              "",
			Name:               "User",
			SubscriptionTier:   "FREE",
			SubscriptionStatus: "ACTIVE",
		}
		if err := s.UserRepo.UpsertUser(ctx, user); err != nil {
			return false, fmt.Errorf("failed to auto-create missing user: %w", err)
		}
	}

	if user.SubscriptionTier != "PRO" || user.SubscriptionStatus != "ACTIVE" {
		return false, nil
	}
	if user.SubscriptionEndsAt != nil && time.Now().After(*user.SubscriptionEndsAt) {
		log.Printf("PRO expired for %s — treating as FREE.", userID)
		return false, nil
	}
	return true, nil
}

// IncrementQuota increments the user's trip quota for the current month
func (s *SubscriptionService) IncrementQuota(ctx context.Context, userID string) error {
	month := time.Now().Format("2006-01")
	return s.SubRepo.IncrementQuota(ctx, userID, month)
}

// HandleMayarPaymentPaid processes a confirmed payment.paid event from Mayar.id.
// referenceID is the user's Clerk ID passed as referenceId during checkout creation.
// mayarPaymentID is stored in the StripeEventID column (repurposed as payment_event_id).
func (s *SubscriptionService) HandleMayarPaymentPaid(ctx context.Context, referenceID, email, mayarPaymentID string) error {
	if referenceID == "" {
		return fmt.Errorf("HandleMayarPaymentPaid: referenceId is empty — cannot identify user")
	}

	// 1. Upgrade user to PRO (no Stripe customer/sub IDs for Mayar — pass empty strings)
	if err := s.UserRepo.UpdateSubscription(ctx, referenceID, "PRO", "ACTIVE", "", ""); err != nil {
		return fmt.Errorf("HandleMayarPaymentPaid: UpdateSubscription failed: %w", err)
	}

	log.Printf("[Mayar] User %s upgraded to PRO (payment: %s)", referenceID, mayarPaymentID)

	// 2. Log subscription event — reuse StripeEventID field as the Mayar payment reference
	return s.SubRepo.LogSubscriptionEvent(ctx, &domain.SubscriptionEvent{
		UserID:        referenceID,
		EventType:     "upgraded",
		FromTier:      "FREE",
		ToTier:        "PRO",
		StripeEventID: mayarPaymentID,
		Metadata:      fmt.Sprintf(`{"provider": "mayar.id", "email": "%s", "payment_id": "%s"}`, email, mayarPaymentID),
	})
}
