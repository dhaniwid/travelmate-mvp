package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"travelmate/internal/domain"

	"github.com/stripe/stripe-go/v78"
)

// UserRepo defines user data access methods
type UserRepo interface {
	GetUserByClerkID(ctx context.Context, clerkID string) (*domain.User, error)
	UpsertUser(ctx context.Context, user *domain.User) error
	UpdateSubscription(ctx context.Context, userID, tier, status, stripeCustID, stripeSubID string) error
	GetUserByStripeID(ctx context.Context, stripeCustID string) (*domain.User, error)
}

// SubRepo defines subscription data access methods
type SubRepo interface {
	GetQuota(ctx context.Context, userID, month string) (*domain.TripQuota, error)
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

// HandleStripeEvent processes webhook events from Stripe
func (s *SubscriptionService) HandleStripeEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		// Handle new subscription via Checkout
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}
		return s.handleCheckoutCompleted(ctx, &session, event.ID)

	case "customer.subscription.updated":
		// Handle status changes (past_due, active)
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		return s.handleSubscriptionUpdated(ctx, &sub, event.ID)

	case "customer.subscription.deleted":
		// Handle cancellation
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		return s.handleSubscriptionDeleted(ctx, &sub, event.ID)
	}

	return nil
}

func (s *SubscriptionService) handleCheckoutCompleted(ctx context.Context, session *stripe.CheckoutSession, eventID string) error {
	userID := session.Metadata["user_id"]
	if userID == "" {
		userID = session.ClientReferenceID
	}
	if userID == "" {
		return fmt.Errorf("no user_id found in session")
	}

	stripeCustID := session.Customer.ID
	stripeSubID := session.Subscription.ID

	// Update User to PRO
	if err := s.UserRepo.UpdateSubscription(ctx, userID, "PRO", "ACTIVE", stripeCustID, stripeSubID); err != nil {
		return err
	}

	// Log Event
	return s.SubRepo.LogSubscriptionEvent(ctx, &domain.SubscriptionEvent{
		UserID:        userID,
		EventType:     "upgraded",
		FromTier:      "FREE",
		ToTier:        "PRO",
		StripeEventID: eventID,
		Metadata:      fmt.Sprintf(`{"reason": "checkout_completed", "session_id": "%s"}`, session.ID),
	})
}

func (s *SubscriptionService) handleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription, eventID string) error {
	stripeCustID := sub.Customer.ID
	stripeSubID := sub.ID
	status := string(sub.Status) // active, past_due, canceled, incomplete

	// 1. Find User by Stripe Customer ID
	user, err := s.UserRepo.GetUserByStripeID(ctx, stripeCustID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found for stripe customer %s", stripeCustID)
	}

	// 2. Map Stripe Status to Internal Status
	internalStatus := "ACTIVE"
	tier := "PRO"
	if status != "active" && status != "trialing" {
		internalStatus = "EXPIRED" // or CANCELLED, depending on logic
		if status == "past_due" {
			internalStatus = "PAST_DUE" // If we supported it, but schema constraint allows limited values
		}
		// Fallback for simple logic: if not active, downgrade to FREE eventually?
		// For now, let's keep it simple:
		if status == "canceled" || status == "unpaid" {
			tier = "FREE"
			internalStatus = "CANCELLED"
		}
	}

	// 3. Update User
	if err := s.UserRepo.UpdateSubscription(ctx, user.UserID, tier, internalStatus, stripeCustID, stripeSubID); err != nil {
		return err
	}

	// 4. Log Event
	return s.SubRepo.LogSubscriptionEvent(ctx, &domain.SubscriptionEvent{
		UserID:        user.UserID,
		EventType:     "status_change",
		FromTier:      user.SubscriptionTier,
		ToTier:        tier,
		StripeEventID: eventID,
		Metadata:      fmt.Sprintf(`{"status": "%s", "stripe_sub_id": "%s"}`, status, stripeSubID),
	})
}

func (s *SubscriptionService) handleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription, eventID string) error {
	stripeCustID := sub.Customer.ID

	// 1. Find User
	user, err := s.UserRepo.GetUserByStripeID(ctx, stripeCustID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found for stripe customer %s", stripeCustID)
	}

	// 2. Downgrade to FREE
	if err := s.UserRepo.UpdateSubscription(ctx, user.UserID, "FREE", "CANCELLED", stripeCustID, ""); err != nil {
		return err
	}

	// 3. Log Event
	return s.SubRepo.LogSubscriptionEvent(ctx, &domain.SubscriptionEvent{
		UserID:        user.UserID,
		EventType:     "cancelled",
		FromTier:      user.SubscriptionTier,
		ToTier:        "FREE",
		StripeEventID: eventID,
		Metadata:      `{"reason": "subscription_deleted"}`,
	})
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

	// Calculate remaining
	quota.Remaining = quota.QuotaLimit - quota.TripsCreated
	if quota.Remaining < 0 {
		quota.Remaining = 0
	}
	quota.IsUnlimited = false

	return quota, nil
}

// CheckQuotaAvailability checks if user can create a trip
func (s *SubscriptionService) CheckQuotaAvailability(ctx context.Context, userID string) (bool, error) {
	// 1. Get user tier
	user, err := s.UserRepo.GetUserByClerkID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, fmt.Errorf("user not found") // Should be created by middleware/auth
	}

	// 2. If PRO, allow
	if user.SubscriptionTier == "PRO" && user.SubscriptionStatus == "ACTIVE" {
		return true, nil
	}

	// 3. If FREE, check quota
	month := time.Now().Format("2006-01")
	quota, err := s.SubRepo.GetQuota(ctx, userID, month)
	if err != nil {
		return false, err
	}

	if quota.TripsCreated >= quota.QuotaLimit {
		return false, nil
	}

	return true, nil
}
