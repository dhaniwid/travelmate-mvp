package domain

import "time"

// UserPreferences represents the global travel DNA for a user
type UserPreferences struct {
	UserID      string    `json:"user_id"`
	Pace        string    `json:"pace"`         // RELAXED, BALANCED, FAST
	BudgetTier  string    `json:"budget_tier"`  // BUDGET, MID, LUXURY
	Dietary     []string  `json:"dietary"`      // JSONB array
	Interests   []string  `json:"interests"`    // JSONB array
	TravelStyle []string  `json:"travel_style"` // JSONB array
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
