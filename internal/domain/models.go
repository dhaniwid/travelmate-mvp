package domain

import (
	"time"
)

// ==========================================
// 1. TRIP & PLAN CORE
// ==========================================

type Trip struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"` // ID dari Clerk
	LocationID  string    `json:"location_id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	StartDate   string    `json:"start_date"` // YYYY-MM-DD
	TripDays    int       `json:"trip_days"`
	Style       string    `json:"style"`        // relaxed, fast, cultural
	BudgetRange string    `json:"budget_range"` // e.g. "2.8-3.2jt"
	Budget      int64     `json:"budget"`       // Changed to int64 for consistency
	IsPublic    bool      `json:"is_public" db:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	PlanData    *TripPlan `json:"plan_data,omitempty" db:"plan_data"`
	Status      string    `json:"status"` // "DRAFT", "UPCOMING", "COMPLETED"
}

type TripPlan struct {
	TripID          string          `json:"trip_id"`
	Itinerary       []ItineraryDay  `json:"itinerary"`
	BudgetBreakdown BudgetBreakdown `json:"budget_breakdown"`
	/// Logistics Section
	LogisticsContext     LogisticsContext      `json:"logistics_context"`
	TransportOptions     []TransportOption     `json:"transport_options"`
	AccommodationOptions []AccommodationOption `json:"strategic_accommodation"`
	DecisionNotes        []string              `json:"decision_notes"`
	PackingList          []PackingItem         `json:"packing_list"` // The new AI feature!
}

type PackingItem struct {
	Category string   `json:"category"` // Contoh: "Clothing", "Toiletries"
	Items    []string `json:"items"`    // Contoh: ["T-Shirt", "Jacket", "Jeans"]
}

type TripAndPlan struct {
	Trip Trip     `json:"trip"`
	Plan TripPlan `json:"plan"`
}

// ==========================================
// 2. ITINERARY DETAILS
// ==========================================

type ItineraryDay struct {
	Day             int             `json:"day"`
	Title           string          `json:"title"`
	MorningBriefing MorningBriefing `json:"morning_briefing"` // <-- NEW: Contextual Intelligence
	Activities      []Activity      `json:"activities"`
}

type MorningBriefing struct {
	WeatherForecast string `json:"weather_forecast"` // e.g. "Sunny, 28°C"
	OutfitTip       string `json:"outfit_tip"`       // e.g. "Wear light cotton"
	LocalVibe       string `json:"local_vibe"`       // e.g. "Busy market day"
}

type Activity struct {
	Time        string `json:"time"`
	Activity    string `json:"activity"`
	Type        string `json:"type"`
	Description string `json:"description"`
	PlaceName   string `json:"place_name"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	TransitTime   string `json:"transit_time"`   // e.g. "15 min"
	TransitMethod string `json:"transit_method"` // e.g. "Walk" or "Taxi"
	TransitPrice  int64  `json:"transit_price"`  // Estimasi biaya transport lokal (IDR)

	Alternatives []ActivityAlternative `json:"alternatives,omitempty"`
}

type ActivityAlternative struct {
	Activity    string `json:"activity"`
	Type        string `json:"type"`
	Description string `json:"description"`
	PlaceName   string `json:"place_name"`
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

// 1. New Helper Structs (Nested Objects)
type TransportBreakdown struct {
	FirstMile string `json:"first_mile"` // e.g., "Taxi to Halim (45m)"
	MainLeg   string `json:"main_leg"`   // e.g., "Whoosh to Padalarang (30m)"
	LastMile  string `json:"last_mile"`  // e.g., "Feeder to City (20m)"
}

type HubDetails struct {
	DepartureNode string `json:"departure_node"` // e.g., "Halim Station"
	ArrivalNode   string `json:"arrival_node"`   // e.g., "Padalarang Station"
}

type LogisticsContext struct {
	DistanceKM   int    `json:"distance_km"`
	RouteType    string `json:"route_type"`    // e.g., "Inter-City" | "Inter-Island"
	WarningAlert string `json:"warning_alert"` // e.g., "Heavy traffic on Friday"
}

// 2. Updated Transport Option
type TransportOption struct {
	StrategyTag          string             `json:"strategy_tag"` // CEPAT | HEMAT
	Name                 string             `json:"name"`
	PriceTier            string             `json:"price_tier"`             // LOW | MED | HIGH
	TotalDurationDisplay string             `json:"total_duration_display"` // Ganti estimated_time
	HubDetails           HubDetails         `json:"hub_details"`
	Breakdown            TransportBreakdown `json:"breakdown"`
	OperatorsHint        string             `json:"operators_hint"` // NEW: "Garuda, Citilink"
	BookingQuery         string             `json:"booking_query"`  // NEW: "flight jakarta to bali"
	Pros                 string             `json:"pros"`
}

// 3. Updated Accommodation Option
type AccommodationOption struct {
	Type                 string `json:"type"`                  // Hotel | Villa
	AreaName             string `json:"area_name"`             // Ganti location_area
	RecommendationReason string `json:"recommendation_reason"` // Ganti location_note
	Vibe                 string `json:"vibe"`                  // Ganti description
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
