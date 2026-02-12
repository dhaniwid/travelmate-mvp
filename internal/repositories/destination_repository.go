package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"travelmate/internal/domain"
)

type DestinationRepository struct {
	DB *sql.DB
}

func NewDestinationRepository(db *sql.DB) *DestinationRepository {
	return &DestinationRepository{DB: db}
}

// GetTrending fetches destinations marked as trending
func (r *DestinationRepository) GetTrending(ctx context.Context, limit int) ([]domain.Destination, error) {
	query := `
		SELECT id, name, country, description, image_url, category, tags, is_trending
		FROM destinations
		WHERE is_trending = TRUE
		LIMIT $1
	`
	rows, err := r.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending destinations: %w", err)
	}
	defer rows.Close()

	var dests []domain.Destination
	for rows.Next() {
		var d domain.Destination
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Country, &d.Description, &d.ImageURL, &d.Category, &d.Tags, &d.IsTrending,
		); err != nil {
			return nil, fmt.Errorf("failed to scan destination: %w", err)
		}
		dests = append(dests, d)
	}
	return dests, nil
}

// GetAll fetches all destinations, optionally filtered by category (if needed later)
func (r *DestinationRepository) GetAll(ctx context.Context) ([]domain.Destination, error) {
	query := `
		SELECT id, name, country, description, image_url, category, tags, is_trending
		FROM destinations
	`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all destinations: %w", err)
	}
	defer rows.Close()

	var dests []domain.Destination
	for rows.Next() {
		var d domain.Destination
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Country, &d.Description, &d.ImageURL, &d.Category, &d.Tags, &d.IsTrending,
		); err != nil {
			return nil, fmt.Errorf("failed to scan destination: %w", err)
		}
		dests = append(dests, d)
	}
	return dests, nil
}

// GetByCategory fetches destinations by category
func (r *DestinationRepository) GetByCategory(ctx context.Context, category string) ([]domain.Destination, error) {
	query := `
		SELECT id, name, country, description, image_url, category, tags, is_trending
		FROM destinations
		WHERE category = $1
	`
	rows, err := r.DB.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch destinations by category: %w", err)
	}
	defer rows.Close()

	var dests []domain.Destination
	for rows.Next() {
		var d domain.Destination
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Country, &d.Description, &d.ImageURL, &d.Category, &d.Tags, &d.IsTrending,
		); err != nil {
			return nil, fmt.Errorf("failed to scan destination: %w", err)
		}
		dests = append(dests, d)
	}
	return dests, nil
}
