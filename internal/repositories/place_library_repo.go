package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"travelmate/internal/domain"
)

type PlaceLibraryRepository struct {
	db *sql.DB
}

func NewPlaceLibraryRepository(db *sql.DB) *PlaceLibraryRepository {
	return &PlaceLibraryRepository{db: db}
}

func (r *PlaceLibraryRepository) GetByName(ctx context.Context, name string) (*domain.PlaceLibraryItem, error) {
	query := `SELECT id, name, google_place_id, description, photos, rating, category, address, latitude, longitude, website, phone, opening_hours, created_at, updated_at 
	          FROM place_library WHERE LOWER(name) = LOWER($1) LIMIT 1`
	var item domain.PlaceLibraryItem
	var photosJSON, openingHoursJSON []byte

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&item.ID, &item.Name, &item.GooglePlaceID, &item.Description, &photosJSON,
		&item.Rating, &item.Category, &item.Address, &item.Latitude, &item.Longitude,
		&item.Website, &item.Phone, &openingHoursJSON, &item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get place by name: %w", err)
	}
	return &item, nil
}

func (r *PlaceLibraryRepository) Upsert(ctx context.Context, item *domain.PlaceLibraryItem) error {
	query := `
		INSERT INTO place_library (
			name, google_place_id, description, photos, rating, category, 
			address, latitude, longitude, website, phone, opening_hours, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW()
		) ON CONFLICT (LOWER(name)) DO UPDATE SET
			google_place_id = EXCLUDED.google_place_id,
			description = EXCLUDED.description,
			photos = EXCLUDED.photos,
			rating = EXCLUDED.rating,
			category = EXCLUDED.category,
			address = EXCLUDED.address,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			website = EXCLUDED.website,
			phone = EXCLUDED.phone,
			opening_hours = EXCLUDED.opening_hours,
			updated_at = NOW()
	`

	// Ensure Photos and OpeningHours are handleable by the driver (sqlx usually handles JSONB with any)
	// If needed, we could marshal to JSON string first, but sqlx usually does this if the driver supports it.

	_, err := r.db.ExecContext(ctx, query,
		item.Name, item.GooglePlaceID, item.Description, item.Photos, item.Rating, item.Category,
		item.Address, item.Latitude, item.Longitude, item.Website, item.Phone, item.OpeningHours,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert place: %w", err)
	}
	return nil
}
