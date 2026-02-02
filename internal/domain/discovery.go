package domain

import (
	"context"
	"encoding/json"
	"time"
)

// ============================================================================
// 1. DISCOVERY RESPONSE (Output dari AI & Input untuk Frontend)
// ============================================================================

// PlaceHighlight: Struktur untuk objek wisata unggulan (Must Visit)
type PlaceHighlight struct {
	Name string `json:"name"`
	Type string `json:"type"` // e.g. "Nature", "Culture", "Urban"
	Hook string `json:"hook"` // Alasan singkat kenapa harus ke sini
}

// CulinarySignature: Struktur untuk makanan khas
type CulinarySignature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tip         string `json:"tip"` // e.g. "Best eaten at night"
}

// HiddenGem: Struktur untuk tempat tersembunyi
type HiddenGem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DiscoveryResponse: Parent struct yang dikembalikan oleh AI Planner
type DiscoveryResponse struct {
	City              string              `json:"city"`
	Tagline           string              `json:"tagline"`
	Vibes             []string            `json:"vibes"` // Array of strings
	Highlights        []PlaceHighlight    `json:"highlights"`
	CulinarySignature []CulinarySignature `json:"culinary_signature"`
	HiddenGem         HiddenGem           `json:"hidden_gem"`
	HistorySnippet    string              `json:"history_snippet"`
}

// ============================================================================
// 2. DATABASE ENTITIES (Struct untuk Mining & Caching)
// ============================================================================

// DestinationMetadata merepresentasikan tabel 'destinations'
// Digunakan saat kita menyimpan hasil mining dari AI ke DB.
type DestinationMetadata struct {
	ID              int             `json:"id" db:"id"`
	CityName        string          `json:"city_name" db:"city_name"`
	CountryName     string          `json:"country_name" db:"country_name"`
	HeroImageURL    string          `json:"hero_image_url" db:"hero_image_url"`
	Description     string          `json:"description" db:"description"`
	PopularityScore int             `json:"popularity_score" db:"popularity_score"`
	DiscoveryData   json.RawMessage `json:"discovery_data" db:"discovery_data"`
}

// CachedRoute merepresentasikan tabel 'cached_logistics'
// Digunakan untuk menyimpan rute agar hemat biaya AI.
type CachedRoute struct {
	ID              int64     `db:"id"`
	OriginCity      string    `db:"origin_city"`
	DestinationCity string    `db:"destination_city"`
	RouteData       []byte    `db:"route_data"` // JSONB disimpan sebagai byte
	CreatedAt       time.Time `db:"created_at"`
	ExpiresAt       time.Time `db:"expires_at"`
}

// ============================================================================
// 3. REPOSITORY INTERFACE (Contract)
// ============================================================================

// DiscoveryRepository adalah kontrak yang harus diimplementasikan oleh layer repository (Postgres).
// Service akan menggunakan interface ini.
type DiscoveryRepository interface {
	// Mining Features
	GetDestinationByCity(ctx context.Context, city string) (*DestinationMetadata, error)
	SaveDestination(ctx context.Context, data DestinationMetadata) error

	// Logistics Caching Features
	GetCachedRoute(ctx context.Context, origin, dest string) (*CachedRoute, error)
	SaveCachedRoute(ctx context.Context, origin, dest string, routeData []byte, expiry time.Duration) error
}
