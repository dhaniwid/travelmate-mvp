package repositories

import (
	"context"
	"database/sql"
)

type FeatureInterestRepository struct {
	db *sql.DB
}

func NewFeatureInterestRepository(db *sql.DB) *FeatureInterestRepository {
	return &FeatureInterestRepository{db: db}
}

func (r *FeatureInterestRepository) SaveInterest(ctx context.Context, userID, featureKey string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO feature_interest (user_id, feature_key) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, featureKey,
	)
	return err
}
