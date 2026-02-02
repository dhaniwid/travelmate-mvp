package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"travelmate/internal/domain"
)

type DiscoveryRepository struct {
	DB *sql.DB
}

func NewDiscoveryRepo(db *sql.DB) *DiscoveryRepository {
	return &DiscoveryRepository{DB: db}
}

// --- Implementasi Destination Mining ---

func (r *DiscoveryRepository) GetDestinationByCity(ctx context.Context, city string) (*domain.DestinationMetadata, error) {
	query := `
        SELECT id, city_name, description, popularity_score, discovery_data 
        FROM destinations 
        WHERE city_name = $1
    `
	var dest domain.DestinationMetadata

	// TAMBAHKAN &dest.DiscoveryData di SCAN
	err := r.DB.QueryRowContext(ctx, query, city).Scan(
		&dest.ID,
		&dest.CityName,
		&dest.Description,
		&dest.PopularityScore,
		&dest.DiscoveryData,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Return nil jika belum ada datanya
	}
	return &dest, err
}

func (r *DiscoveryRepository) SaveDestination(ctx context.Context, data domain.DestinationMetadata) error {
	query := `
        INSERT INTO destinations (city_name, country_name, description, discovery_data, popularity_score)
        VALUES ($1, 'Indonesia', $2, $3, 1)
        ON CONFLICT (city_name) 
        DO UPDATE SET 
            popularity_score = destinations.popularity_score + 1,
            discovery_data = $3
    `
	_, err := r.DB.ExecContext(ctx, query, data.CityName, data.Description, data.DiscoveryData)
	return err
}

// --- Implementasi Logistics Caching ---

func (r *DiscoveryRepository) GetCachedRoute(ctx context.Context, origin, dest string) (*domain.CachedRoute, error) {
	// Cari rute yang belum expired
	query := `
        SELECT id, route_data, expires_at 
        FROM cached_logistics 
        WHERE origin_city = $1 AND destination_city = $2 AND expires_at > NOW()
        LIMIT 1
    `
	var route domain.CachedRoute
	err := r.DB.QueryRowContext(ctx, query, origin, dest).Scan(&route.ID, &route.RouteData, &route.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &route, err
}

func (r *DiscoveryRepository) SaveCachedRoute(ctx context.Context, origin, dest string, routeData []byte, expiry time.Duration) error {
	query := `
        INSERT INTO cached_logistics (origin_city, destination_city, route_data, expires_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (origin_city, destination_city) -- Pastikan ada Unique constraint di DB
        DO UPDATE SET route_data = $3, expires_at = $4, created_at = NOW()
    `
	expTime := time.Now().Add(expiry)
	_, err := r.DB.ExecContext(ctx, query, origin, dest, routeData, expTime)
	return err
}
