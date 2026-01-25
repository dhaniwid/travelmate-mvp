package repositories

import (
	"context"
	"database/sql"
	"time"
	"travelmate/internal/domain"
)

type RouteRepository struct {
	DB *sql.DB
}

func NewRouteRepository(db *sql.DB) *RouteRepository {
	return &RouteRepository{DB: db}
}

// 1. Cari Rute di DB
func (r *RouteRepository) GetRoute(ctx context.Context, origin, dest, mode string) (*domain.Route, error) {
	query := `
		SELECT id, origin_code, destination_code, transport_mode, avg_duration_mins, last_updated_at
		FROM routes
		WHERE origin_code = $1 AND destination_code = $2 AND transport_mode = $3
	`
	row := r.DB.QueryRowContext(ctx, query, origin, dest, mode)

	var route domain.Route
	err := row.Scan(&route.ID, &route.OriginCode, &route.DestinationCode, &route.TransportMode, &route.AvgDurationMins, &route.LastUpdatedAt)
	if err != nil {
		return nil, err
	}
	return &route, nil
}

// 2. Simpan Rute (Insert or Update)
func (r *RouteRepository) UpsertRoute(ctx context.Context, route domain.Route) error {
	query := `
		INSERT INTO routes (id, origin_code, destination_code, transport_mode, avg_duration_mins, last_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (origin_code, destination_code, transport_mode)
		DO UPDATE SET 
			avg_duration_mins = EXCLUDED.avg_duration_mins,
			last_updated_at = NOW()
	`
	_, err := r.DB.ExecContext(ctx, query, route.ID, route.OriginCode, route.DestinationCode, route.TransportMode, route.AvgDurationMins, time.Now())
	return err
}
