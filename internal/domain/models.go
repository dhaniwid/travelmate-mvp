package domain

import (
	"time"
)

// ==========================================
// 1. TRIP & PLAN CORE
// ==========================================

type Trip struct {
	ID          string    `json:"id"`
	LocationID  string    `json:"location_id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	StartDate   string    `json:"start_date"` // YYYY-MM-DD
	TripDays    int       `json:"trip_days"`
	Style       string    `json:"style"`        // relaxed, fast, cultural
	BudgetRange string    `json:"budget_range"` // e.g. "2.8-3.2jt"
	Budget      int64     `json:"budget"`       // Changed to int64 for consistency
	CreatedAt   time.Time `json:"created_at"`
}

type TripPlan struct {
	TripID               string                `json:"trip_id"`
	Itinerary            []ItineraryDay        `json:"itinerary"`
	BudgetBreakdown      BudgetBreakdown       `json:"budget_breakdown"`
	TransportOptions     []TransportOption     `json:"transport_options"`     // Renamed to Singular
	AccommodationOptions []AccommodationOption `json:"accommodation_options"` // Renamed to Singular
	DecisionNotes        []string              `json:"decision_notes"`
}

type TripAndPlan struct {
	Trip Trip     `json:"trip"`
	Plan TripPlan `json:"plan"`
}

// ==========================================
// 2. ITINERARY DETAILS
// ==========================================

// ItineraryDay: Mewakili satu hari perjalanan
type ItineraryDay struct {
	Day        int        `json:"day"`
	Title      string     `json:"title"`
	Activities []Activity `json:"activities"` // Upgraded from []string to []Activity
}

// Activity: Detail per aktivitas (Sesuai rencana enrichment)
type Activity struct {
	Time        string  `json:"time"`        // "09:00 - 10:00"
	Activity    string  `json:"activity"`    // Nama aktivitas
	Type        string  `json:"type"`        // Culinary, Sightseeing, etc
	Description string  `json:"description"` // Penjelasan singkat
	PlaceName   string  `json:"place_name"`  // Nama tempat (untuk Google Maps)
	Latitude    float64 `json:"latitude"`    // Contoh: -6.9175
	Longitude   float64 `json:"longitude"`   // Contoh: 107.6191
}

type TouristAttraction struct {
	ID              string    `json:"id"`
	LocationID      string    `json:"location_id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
	PopularityScore int       `json:"popularity_score"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ==========================================
// 3. OPTIONS & BREAKDOWN (Value Objects)
// ==========================================

type BudgetBreakdown struct {
	Transport     FlexibleInt64 `json:"transport"`
	Accommodation FlexibleInt64 `json:"accommodation"`
	Food          FlexibleInt64 `json:"food"`
	Tickets       FlexibleInt64 `json:"tickets"`
	Misc          FlexibleInt64 `json:"misc"`
}

// TransportOption: Struktur untuk opsi transport dari AI
type TransportOption struct {
	Type          string `json:"type"` // Flight, Train, Bus
	Name          string `json:"name"` // Garuda, Whoosh
	Price         int64  `json:"price"`
	EstimatedTime string `json:"estimated_time"`
	Pros          string `json:"pros"`
}

// AccommodationOption: Struktur untuk opsi hotel dari AI
type AccommodationOption struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Rating        string `json:"rating"`
	PricePerNight int64  `json:"price_per_night"`
	LocationArea  string `json:"location_area"`
	Description   string `json:"description"`
	LocationNote  string `json:"location_note"`
	ImageURL      string `json:"image_url"`
}

// ==========================================
// 4. AI PARSING STRUCTS
// ==========================================

// AIPlannerResponse: Struct bayangan untuk menangkap Raw JSON dari AI Planner
type AIPlannerResponse struct {
	Itinerary            []ItineraryDay        `json:"itinerary"`
	BudgetBreakdown      BudgetBreakdown       `json:"budget_breakdown"` // Fixed: Object, not Array
	TransportOptions     []TransportOption     `json:"transport_options"`
	AccommodationOptions []AccommodationOption `json:"accommodation_options"`
	DecisionNotes        []string              `json:"decision_notes"`
}

type LocationMetadataResponse struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
	Styles      []string `json:"styles"`
	HubType     string   `json:"hub_type"` // "airport" or "station"
	HubCode     string   `json:"hub_code"` // "TJQ"
	HubName     string   `json:"hub_name"`
}

// ==========================================
// 5. DATABASE ENTITIES (Persistent Data)
// ==========================================

type Location struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Country      string       `json:"country"`
	Description  string       `json:"description"`
	ImageURL     string       `json:"image_url"`
	StyleTags    []string     `json:"style_tags"`
	TransportHub TransportHub `json:"transport_hub"`
}

type TransportHub struct {
	ID          string `json:"id"`
	LocationID  string `json:"location_id"`
	Type        string `json:"type"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Coordinates string `json:"coordinates"` // New field for lat,long
}

type Accommodation struct {
	ID            string    `json:"id"`
	LocationID    string    `json:"location_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Rating        string    `json:"rating"`
	PricePerNight int64     `json:"price_per_night"`
	LocationArea  string    `json:"location_area"` // Mapped to 'address' in DB
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	LastUpdated   time.Time `json:"last_updated"`
}

type Route struct {
	ID              string    `json:"id"`
	OriginCode      string    `json:"origin_code"`
	DestinationCode string    `json:"destination_code"`
	TransportMode   string    `json:"transport_mode"`
	ProviderName    string    `json:"provider_name"`
	Price           int64     `json:"price"`
	AvgDurationMins int       `json:"avg_duration_mins"`
	LastUpdatedAt   time.Time `json:"last_updated_at"`
}

type RoutePrice struct {
	ID          string    `json:"id"`
	RouteID     string    `json:"route_id"`
	Provider    string    `json:"provider"`
	PriceAmount int64     `json:"price_amount"` // Gunakan int64
	TravelDate  string    `json:"travel_date"`  // YYYY-MM-DD
	RecordedAt  time.Time `json:"recorded_at"`
}

type Feedback struct {
	ID        string    `json:"id"`
	TripID    string    `json:"trip_id"`
	Rating    int       `json:"rating"` // Changed to int (1-5) to match DB
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Utilities ---

// TripRequest: (Opsional) Jika ingin memisahkan input user dengan entity Trip
type TripRequest struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	TripDays    int    `json:"trip_days"`
	Budget      int64  `json:"budget"`
	Style       string `json:"style"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ==========================================
// 6. PERFORMANCE METRICS
// ==========================================

type PerformanceMetric struct {
	ID          string    `json:"id"`
	TaskName    string    `json:"task_name"`
	DurationMS  int       `json:"duration_ms"`
	Destination string    `json:"destination"`
	ModelUsed   string    `json:"model_used"`
	CreatedAt   time.Time `json:"created_at"`
}

type PerformanceStats struct {
	TaskName   string  `json:"task_name"`
	AvgLatency float64 `json:"avg_latency"`
	MaxLatency int     `json:"max_latency"`
	TotalCalls int     `json:"total_calls"`
}
