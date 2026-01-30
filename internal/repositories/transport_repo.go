package repositories

import (
	"context"
	"database/sql"
	"log"
	"time"
	"travelmate/internal/domain"

	"github.com/google/uuid"
)

type TransportRepository struct {
	DB *sql.DB
}

func NewTransportRepository(db *sql.DB) *TransportRepository {
	return &TransportRepository{DB: db}
}

// SaveRoute: Menyimpan data rute (Learning) dengan Log yang jelas
func (r *TransportRepository) SaveRoute(ctx context.Context, route domain.Route) error {
	// Pastikan ID terisi (UUID String)
	if route.ID == "" {
		route.ID = uuid.New().String()
	}

	// SQL Query yang SUDAH DISESUAIKAN dengan tabel terakhir
	query := `
		INSERT INTO routes (
			id, origin_code, destination_code, transport_mode, provider_name, 
			price, avg_duration_mins, last_updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (origin_code, destination_code, transport_mode) 
		DO UPDATE SET 
			price = EXCLUDED.price,
			avg_duration_mins = EXCLUDED.avg_duration_mins,
			provider_name = EXCLUDED.provider_name,
			last_updated_at = EXCLUDED.last_updated_at
	`

	_, err := r.DB.ExecContext(ctx, query,
		route.ID,
		route.OriginCode,
		route.DestinationCode,
		route.TransportMode, // FIX: Mode
		route.ProviderName,
		route.Price,
		route.AvgDurationMins, // FIX: Int
		time.Now(),
	)

	if err != nil {
		log.Printf("❌ [Repo] Failed to save route %s->%s: %v", route.OriginCode, route.DestinationCode, err)
		return err
	}

	log.Printf("💾 [Repo] Saved/Updated route cache: %s -> %s (%s)", route.OriginCode, route.DestinationCode, route.TransportMode)
	return nil
}

// GetRoute: Mencari rute di database
func (r *TransportRepository) GetRoute(ctx context.Context, origin, destination string) ([]domain.Route, error) {
	query := `
		SELECT id, origin_code, destination_code, transport_mode, provider_name, price, avg_duration_mins, last_updated_at
		FROM routes 
		WHERE origin_code = $1 AND destination_code = $2
	`

	rows, err := r.DB.QueryContext(ctx, query, origin, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []domain.Route
	for rows.Next() {
		var rt domain.Route
		if err := rows.Scan(
			&rt.ID,
			&rt.OriginCode,
			&rt.DestinationCode,
			&rt.TransportMode,
			&rt.ProviderName,
			&rt.Price,
			&rt.AvgDurationMins,
			&rt.LastUpdatedAt,
		); err != nil {
			return nil, err
		}
		routes = append(routes, rt)
	}

	return routes, nil
}

func (r *TransportRepository) SavePrice(ctx context.Context, price domain.RoutePrice) error {
	query := `
       INSERT INTO route_prices (id, route_id, provider, price_amount, travel_date, recorded_at)
       VALUES ($1, $2, $3, $4, $5, $6)
    `

	_, err := r.DB.ExecContext(ctx, query,
		price.ID,
		price.RouteID,
		price.Provider,
		price.PriceAmount,
		price.TravelDate,
		time.Now(),
	)
	return err
}

// UpsertTransportOption menyimpan atau update data transport
//func (r *TransportRepository) UpsertTransportOption(ctx context.Context, opt domain.TransportOption, origin, dest string) error {
//	query := `
//        INSERT INTO transport_seed_options
//        (origin_city, destination_city, transport_type, provider_name, estimated_price_min, estimated_duration, pros, created_at)
//        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
//        ON CONFLICT (origin_city, destination_city, provider_name, transport_type)
//        DO UPDATE SET
//            estimated_price_min = EXCLUDED.estimated_price_min,
//            estimated_duration = EXCLUDED.estimated_duration,
//            pros = EXCLUDED.pros,
//            created_at = NOW() -- Refresh timestamp agar tau ini data baru
//    `
//	// Convert FlexibleInt64 ke int64 standard atau float untuk DB
//	price := int64(opt.Price)
//
//	_, err := r.DB.ExecContext(ctx, query, origin, dest, opt.Type, opt.Name, price, opt.EstimatedTime, opt.Pros)
//	return err
//}
