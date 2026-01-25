package repositories

import (
	"context"
	"database/sql"
	"travelmate/internal/domain"
)

type AttractionRepository struct {
	DB *sql.DB
}

func NewAttractionRepository(db *sql.DB) *AttractionRepository {
	return &AttractionRepository{DB: db}
}

// UpsertAttraction melakukan simpan data jika belum ada, atau update score jika sudah ada.
func (r *AttractionRepository) UpsertAttraction(ctx context.Context, attr domain.TouristAttraction) error {
	query := `
		INSERT INTO tourist_attractions (
			id, location_id, name, category, description, popularity_score, last_updated
		)
		VALUES ($1, $2, $3, $4, $5, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (name, location_id) 
		DO UPDATE SET 
			popularity_score = tourist_attractions.popularity_score + 1,
			description = EXCLUDED.description, -- Memperbarui deskripsi dengan versi AI terbaru
			last_updated = CURRENT_TIMESTAMP;
	`

	_, err := r.DB.ExecContext(ctx, query,
		attr.ID,
		attr.LocationID,
		attr.Name,
		attr.Category,
		attr.Description,
	)

	return err
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
		if err := rows.Scan(&a.ID, &a.LocationID, &a.Name, &a.Category, &a.Description, &a.PopularityScore, &a.LastUpdated); err != nil {
			return nil, err
		}
		attractions = append(attractions, a)
	}

	return attractions, nil
}
