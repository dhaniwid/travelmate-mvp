package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"travelmate/internal/domain"
)

type AttractionRepository struct {
	DB *sql.DB
}

func NewAttractionRepository(db *sql.DB) *AttractionRepository {
	return &AttractionRepository{DB: db}
}

// UpsertAttraction updates or inserts an attraction into the cache.
func (r *AttractionRepository) UpsertAttraction(ctx context.Context, attr domain.TouristAttraction) error {
	photosJSON, _ := json.Marshal(attr.Photos)
	query := `
		INSERT INTO tourist_attractions (
			id, location_id, name, category, description, 
			latitude, longitude, place_id, photos, visit_duration,
			popularity_score, last_updated
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (name, location_id) 
		DO UPDATE SET 
			popularity_score = tourist_attractions.popularity_score + 1,
			description = EXCLUDED.description, 
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			place_id = EXCLUDED.place_id,
			photos = EXCLUDED.photos,
			visit_duration = EXCLUDED.visit_duration,
			last_updated = CURRENT_TIMESTAMP;
	`

	_, err := r.DB.ExecContext(ctx, query,
		attr.ID,
		attr.LocationID,
		attr.Name,
		attr.Category,
		attr.Description,
		attr.Latitude,
		attr.Longitude,
		attr.PlaceID,
		photosJSON,
		attr.VisitDuration,
	)

	return err
}

// GetByName retrieves an attraction by its name and location ID from the cache.
func (r *AttractionRepository) GetByName(ctx context.Context, name string, locationID string) (*domain.TouristAttraction, error) {
	query := `
		SELECT id, location_id, name, category, description, 
		       latitude, longitude, place_id, photos, visit_duration,
		       popularity_score, last_updated
		FROM tourist_attractions
		WHERE LOWER(name) = LOWER($1) AND location_id = $2
	`

	var attr domain.TouristAttraction
	var photosJSON []byte
	err := r.DB.QueryRowContext(ctx, query, name, locationID).Scan(
		&attr.ID, &attr.LocationID, &attr.Name, &attr.Category, &attr.Description,
		&attr.Latitude, &attr.Longitude, &attr.PlaceID, &photosJSON, &attr.VisitDuration,
		&attr.PopularityScore, &attr.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(photosJSON, &attr.Photos)
	return &attr, nil
}

// GetTopAttractions mengambil daftar tempat wisata paling populer di suatu lokasi.
func (r *AttractionRepository) GetTopAttractions(ctx context.Context, locationID string, limit int) ([]domain.TouristAttraction, error) {
	query := `
		SELECT id, location_id, name, category, description, popularity_score, last_updated
		FROM tourist_attractions
		WHERE location_id = $1
		ORDER BY popularity_score DESC
		LIMIT $2
	`

	rows, err := r.DB.QueryContext(ctx, query, locationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attractions []domain.TouristAttraction
	for rows.Next() {
		var a domain.TouristAttraction
		var photosJSON []byte
		if err := rows.Scan(
			&a.ID, &a.LocationID, &a.Name, &a.Category, &a.Description,
			&a.Latitude, &a.Longitude, &a.PlaceID, &photosJSON, &a.VisitDuration,
			&a.PopularityScore, &a.LastUpdated,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(photosJSON, &a.Photos)
		attractions = append(attractions, a)
	}

	return attractions, nil
}
