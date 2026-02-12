package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"travelmate/internal/domain"
)

type PreferencesRepository struct {
	DB *sql.DB
}

func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
	return &PreferencesRepository{DB: db}
}

// GetPreferences fetches the user's travel DNA
func (r *PreferencesRepository) GetPreferences(ctx context.Context, userID string) (*domain.UserPreferences, error) {
	query := `
		SELECT 
			user_id, pace, budget_tier, dietary, interests, travel_style, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`

	var prefs domain.UserPreferences
	var dietaryJSON, interestsJSON, styleJSON []byte

	err := r.DB.QueryRowContext(ctx, query, userID).Scan(
		&prefs.UserID, &prefs.Pace, &prefs.BudgetTier,
		&dietaryJSON, &interestsJSON, &styleJSON,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Return nil if no preferences set yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch preferences: %w", err)
	}

	// Unmarshal JSONB columns
	if len(dietaryJSON) > 0 {
		_ = json.Unmarshal(dietaryJSON, &prefs.Dietary)
	}
	if len(interestsJSON) > 0 {
		_ = json.Unmarshal(interestsJSON, &prefs.Interests)
	}
	if len(styleJSON) > 0 {
		_ = json.Unmarshal(styleJSON, &prefs.TravelStyle)
	}

	return &prefs, nil
}

// UpsertPreferences creates or updates the user's travel DNA
func (r *PreferencesRepository) UpsertPreferences(ctx context.Context, prefs *domain.UserPreferences) error {
	query := `
		INSERT INTO user_preferences (user_id, pace, budget_tier, dietary, interests, travel_style, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			pace = EXCLUDED.pace,
			budget_tier = EXCLUDED.budget_tier,
			dietary = EXCLUDED.dietary,
			interests = EXCLUDED.interests,
			travel_style = EXCLUDED.travel_style,
			updated_at = NOW()
	`

	dietaryJSON, _ := json.Marshal(prefs.Dietary)
	interestsJSON, _ := json.Marshal(prefs.Interests)
	styleJSON, _ := json.Marshal(prefs.TravelStyle)

	_, err := r.DB.ExecContext(ctx, query,
		prefs.UserID,
		prefs.Pace,
		prefs.BudgetTier,
		dietaryJSON,
		interestsJSON,
		styleJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert preferences: %w", err)
	}

	return nil
}
