package repositories

import (
	"context"
	"database/sql"
	"travelmate/internal/domain"
)

type FeedbackRepository struct {
	DB *sql.DB
}

func NewFeedbackRepository(db *sql.DB) *FeedbackRepository {
	return &FeedbackRepository{DB: db}
}

func (r *FeedbackRepository) CreateFeedback(ctx context.Context, fb domain.Feedback) error {
	query := `INSERT INTO feedback (id, trip_id, rating, comment, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.DB.ExecContext(ctx, query, fb.ID, fb.TripID, fb.Rating, fb.Comment, fb.CreatedAt)
	return err
}
