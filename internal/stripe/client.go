package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/webhook"
)

type Client struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

func NewClient(secretKey, webhookSecret string) *Client {
	stripe.Key = secretKey
	return &Client{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
		// TODO: Move URLs to config
		SuccessURL: "http://localhost:3000/dashboard?subscription_success=true",
		CancelURL:  "http://localhost:3000/pricing?canceled=true",
	}
}

// CreateCheckoutSession creates a new Stripe Checkout Session for subscription
func (c *Client) CreateCheckoutSession(userID, email, priceID string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:        stripe.String(c.SuccessURL),
		CancelURL:         stripe.String(c.CancelURL),
		CustomerEmail:     stripe.String(email),
		ClientReferenceID: stripe.String(userID),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": userID,
			},
		},
		Metadata: map[string]string{
			"user_id": userID, // Also on session metadata for easier lookup
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create checkout session: %w", err)
	}

	return sess.URL, nil
}

// ConstructEvent validates and parses webhook events
func (c *Client) ConstructEvent(payload []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, signature, c.WebhookSecret)
}
