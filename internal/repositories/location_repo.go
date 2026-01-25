package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"travelmate/internal/domain"
)

type LocationRepository struct {
	DB *sql.DB
}

func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{DB: db}
}

func (r *LocationRepository) FindByName(ctx context.Context, name string) (*domain.Location, error) {
	query := `
       SELECT l.id, l.name, l.country, l.description
       FROM locations l
       WHERE l.name = $1
       LIMIT 1
    `
	row := r.DB.QueryRowContext(ctx, query, name)

	var loc domain.Location
	var styles []string

	err := row.Scan(
		&loc.ID, &loc.Name, &loc.Country, &loc.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Benar-benar tidak ada data
		}
		return nil, fmt.Errorf("scan error: %v", err) // Ada data tapi gagal baca
	}

	loc.StyleTags = styles

	return &loc, nil
}

func (r *LocationRepository) SaveLocation(ctx context.Context, loc domain.Location) error {
	tagsJSON, err := json.Marshal(loc.StyleTags)
	if err != nil {
		tagsJSON = []byte("[]")
		log.Printf("⚠️ Failed to marshal styles: %v", err)
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// 1. Insert Location
	queryLoc := `INSERT INTO locations (id, name, country, description, style_tags, hub_type, hub_code, hub_name, image_url) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	             ON CONFLICT (name) DO NOTHING` // Safety net
	_, err = tx.ExecContext(ctx, queryLoc, loc.ID, loc.Name, loc.Country, loc.Description, tagsJSON,
		loc.TransportHub.Type, loc.TransportHub.Code, loc.TransportHub.Name, loc.ImageURL)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Insert Hub
	queryHub := `INSERT INTO transport_hubs (id, location_id, type, code, name, city, country) VALUES ($1, $2, $3, $4, $5, $6, $7)
	             ON CONFLICT (code) DO NOTHING`
	_, err = tx.ExecContext(ctx, queryHub, loc.TransportHub.ID, loc.ID, loc.TransportHub.Type,
		loc.TransportHub.Code, loc.TransportHub.Name, loc.TransportHub.City, loc.TransportHub.Country)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
