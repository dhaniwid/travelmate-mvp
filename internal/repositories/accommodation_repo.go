package repositories

import (
	"context"
	"database/sql"
	"log"
	"time"
	"travelmate/internal/domain"

	"github.com/google/uuid"
)

type AccommodationRepository struct {
	DB *sql.DB
}

func NewAccommodationRepository(db *sql.DB) *AccommodationRepository {
	return &AccommodationRepository{DB: db}
}

func (r *AccommodationRepository) SaveAccommodation(ctx context.Context, acc domain.Accommodation) error {
	if acc.ID == "" {
		acc.ID = uuid.New().String()
	}

	query := `
		INSERT INTO accommodations (
			id, location_id, name, type, rating, 
			price_per_night, address, description, last_updated
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (location_id, name) 
		DO UPDATE SET 
			type = EXCLUDED.type,
			rating = EXCLUDED.rating,
			price_per_night = EXCLUDED.price_per_night,
			address = EXCLUDED.address,
			description = EXCLUDED.description,
			last_updated = EXCLUDED.last_updated
	`

	_, err := r.DB.ExecContext(ctx, query,
		acc.ID,
		acc.LocationID,
		acc.Name,
		acc.Type,
		acc.Rating,
		acc.PricePerNight,
		acc.LocationArea,
		acc.Description,
		time.Now(),
	)

	if err != nil {
		log.Printf("❌ [AccomRepo] Failed to save hotel '%s': %v", acc.Name, err)
		return err
	}

	return nil
}
