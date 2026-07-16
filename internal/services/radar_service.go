package services

import (
	"context"
	"database/sql"
	"log"
	"math"
	"time"
	"travelmate/internal/domain"
)

// RadarNearbyItem represents a single POI returned by the radar query.
type RadarNearbyItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	DistanceMeters  float64 `json:"distance_meters"`
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	HasStamp        bool    `json:"has_stamp"`
	Slug            string  `json:"slug"`
	LandmarkSlug    string  `json:"landmark_slug,omitempty"`
}

// RadarLocation is the reverse-geocoded area info.
type RadarLocation struct {
	Area     string `json:"area"`
	City     string `json:"city"`
	Province string `json:"province"`
}

// RadarResponse is the full /radar response.
type RadarResponse struct {
	Location   RadarLocation     `json:"location"`
	Nearby     []RadarNearbyItem `json:"nearby"`
	ActiveTrip *domain.Trip      `json:"active_trip"`
}

type RadarService struct {
	DB *sql.DB
}

func NewRadarService(db *sql.DB) *RadarService {
	return &RadarService{DB: db}
}

// GetRadar queries POIs within radius meters of (lat, lng) for the given user.
func (s *RadarService) GetRadar(ctx context.Context, lat, lng float64, radiusMeters int, userID string) (*RadarResponse, error) {
	if radiusMeters <= 0 || radiusMeters > 50000 {
		radiusMeters = 1000
	}

	// 1. Geospatial query — Haversine in SQL (no PostGIS required)
	rows, err := s.DB.QueryContext(ctx, `
		SELECT
			id, name, category, description,
			has_landmark_svg, landmark_slug,
			(6371000 * acos(
				LEAST(1.0,
					cos(radians($1)) * cos(radians(lat)) * cos(radians(lng) - radians($2))
					+ sin(radians($1)) * sin(radians(lat))
				)
			)) AS distance_meters
		FROM local_knowledge
		WHERE lat IS NOT NULL AND lng IS NOT NULL
		  AND lat BETWEEN $1 - ($3::float / 111320)
		  AND $1 + ($3::float / 111320)
		  AND lng BETWEEN $2 - ($3::float / (111320 * cos(radians($1))))
		  AND $2 + ($3::float / (111320 * cos(radians($1))))
		ORDER BY distance_meters
		LIMIT 20
	`, lat, lng, float64(radiusMeters))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nearby []RadarNearbyItem
	for rows.Next() {
		var item RadarNearbyItem
		var hasLandmark bool
		var landmarkSlug sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Description,
			&hasLandmark, &landmarkSlug, &item.DistanceMeters); err != nil {
			log.Printf("⚠️ [RADAR] scan error: %v", err)
			continue
		}
		item.HasStamp = hasLandmark
		if landmarkSlug.Valid {
			item.LandmarkSlug = landmarkSlug.String
			item.Slug = landmarkSlug.String
		}
		// Round distance to integer meters
		item.DistanceMeters = math.Round(item.DistanceMeters)
		nearby = append(nearby, item)
	}
	if nearby == nil {
		nearby = []RadarNearbyItem{}
	}

	// 2. Reverse geocode — hardcoded city lookup for MVP
	loc := reverseGeocodeMVP(lat, lng)

	// 3. Active trip — most recent in-progress trip for signed-in users
	var activeTrip *domain.Trip
	if userID != "" && userID != "guest" {
		activeTrip = s.getActiveTrip(ctx, userID)
	}

	return &RadarResponse{
		Location:   loc,
		Nearby:     nearby,
		ActiveTrip: activeTrip,
	}, nil
}

// getActiveTrip returns the user's most recent active trip (status != DRAFT, within travel window).
func (s *RadarService) getActiveTrip(ctx context.Context, userID string) *domain.Trip {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var t domain.Trip
	err := s.DB.QueryRowContext(ctx2, `
		SELECT id, destination, start_date, trip_days, status
		FROM trips
		WHERE user_id = $1
		  AND status NOT IN ('DRAFT', 'DELETED')
		  AND start_date IS NOT NULL
		  AND start_date <= NOW()
		  AND (start_date + (trip_days * INTERVAL '1 day')) >= NOW()
		ORDER BY start_date DESC
		LIMIT 1
	`, userID).Scan(&t.ID, &t.Destination, &t.StartDate, &t.TripDays, &t.Status)
	if err != nil {
		return nil
	}
	return &t
}

// ── City bounding boxes for MVP reverse geocode ───────────────────────────────

type cityBounds struct {
	Area     string
	City     string
	Province string
	MinLat   float64
	MaxLat   float64
	MinLng   float64
	MaxLng   float64
}

var mvpCityBounds = []cityBounds{
	{Area: "Kota Jakarta", City: "Jakarta", Province: "DKI Jakarta", MinLat: -6.37, MaxLat: -6.10, MinLng: 106.65, MaxLng: 107.00},
	{Area: "Kuta", City: "Bali", Province: "Bali", MinLat: -8.75, MaxLat: -8.58, MinLng: 115.10, MaxLng: 115.25},
	{Area: "Seminyak", City: "Bali", Province: "Bali", MinLat: -8.70, MaxLat: -8.63, MinLng: 115.14, MaxLng: 115.18},
	{Area: "Ubud", City: "Bali", Province: "Bali", MinLat: -8.53, MaxLat: -8.48, MinLng: 115.24, MaxLng: 115.29},
	{Area: "Malioboro", City: "Yogyakarta", Province: "DI Yogyakarta", MinLat: -7.81, MaxLat: -7.78, MinLng: 110.35, MaxLng: 110.38},
	{Area: "Kota Tua", City: "Yogyakarta", Province: "DI Yogyakarta", MinLat: -7.83, MaxLat: -7.78, MinLng: 110.35, MaxLng: 110.40},
	{Area: "Kota Semarang", City: "Semarang", Province: "Jawa Tengah", MinLat: -7.06, MaxLat: -6.95, MinLng: 110.38, MaxLng: 110.46},
	{Area: "Marina Bay", City: "Singapore", Province: "Singapore", MinLat: 1.27, MaxLat: 1.32, MinLng: 103.84, MaxLng: 103.88},
	{Area: "Shibuya", City: "Tokyo", Province: "Tokyo", MinLat: 35.65, MaxLat: 35.67, MinLng: 139.69, MaxLng: 139.72},
}

func reverseGeocodeMVP(lat, lng float64) RadarLocation {
	for _, b := range mvpCityBounds {
		if lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng {
			return RadarLocation{Area: b.Area, City: b.City, Province: b.Province}
		}
	}
	// Fallback: return generic based on rough country bbox
	city := guessCityFallback(lat, lng)
	return RadarLocation{Area: city, City: city, Province: ""}
}

func guessCityFallback(lat, lng float64) string {
	// Indonesia rough bbox
	if lat >= -11 && lat <= 6 && lng >= 95 && lng <= 141 {
		return "Indonesia"
	}
	parts := []string{}
	if lat >= 0 { parts = append(parts, "N") } else { parts = append(parts, "S") }
	_ = parts
	return "Unknown Location"
}

