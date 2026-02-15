package domain

import (
	"time"
)

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ==========================================
// 1. TRIP & PLAN CORE
// ==========================================

type Trip struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"` // ID dari Clerk
	LocationID       string    `json:"location_id"`
	Origin           string    `json:"origin"`
	Destination      string    `json:"destination"`
	StartDate        string    `json:"start_date"` // YYYY-MM-DD
	TripDays         int       `json:"trip_days"`
	Style            string    `json:"style"`        // relaxed, fast, cultural
	BudgetRange      string    `json:"budget_range"` // e.g. "2.8-3.2jt"
	Budget           int64     `json:"budget"`       // Changed to int64 for consistency
	IsPublic         bool      `json:"is_public" db:"is_public"`
	CreatedAt        time.Time `json:"created_at"`
	PlanData         *TripPlan `json:"plan_data,omitempty" db:"plan_data"`
	Status           string    `json:"status"`            // "DRAFT", "UPCOMING", "COMPLETED"
	EnrichmentStatus string    `json:"enrichment_status"` // "pending", "enriching", "completed"
	ItineraryStatus  string    `json:"itinerary_status"`  // "pending", "generating", "completed"
	AIEditsUsed      int       `json:"ai_edits_used" db:"ai_edits_used"`
}

const (
	EnrichmentStatusPending   = "pending"
	EnrichmentStatusEnriching = "enriching"
	EnrichmentStatusCompleted = "completed"

	ItineraryStatusPending    = "pending"
	ItineraryStatusGenerating = "generating"
	ItineraryStatusCompleted  = "completed"
)

type TripPlan struct {
	TripID          string          `json:"trip_id"`
	Itinerary       []ItineraryDay  `json:"itinerary"`
	BudgetBreakdown BudgetBreakdown `json:"budget_breakdown"`
	/// Logistics Section
	LogisticsContext     *LogisticsContext     `json:"logistics_context"`
	TransportOptions     []TransportOption     `json:"transport_options"`
	AccommodationOptions []AccommodationOption `json:"strategic_accommodation"`
	DecisionNotes        []string              `json:"decision_notes"`
	PackingList          []PackingCategory     `json:"packing_list"`  // Updated type
	ArrivalGuide         *ArrivalGuide         `json:"arrival_guide"` // New Field
	MorningBriefing      string                `json:"morning_briefing"`
	Highlights           []TripHighlight       `json:"highlights"`
	Logistics            *LogisticsData        `json:"logistics"` // NEW: M-124

	// --- Discovery Features (Merged from DiscoveryView) ---
	Tagline           string              `json:"tagline"`
	Vibes             []string            `json:"vibes"`
	CulinarySignature []CulinarySignature `json:"culinary_signature"`
	HiddenGem         *HiddenGem          `json:"hidden_gem"`
	HistorySnippet    string              `json:"history_snippet"`
}

type LogisticsData struct {
	ArrivalGuide *ArrivalGuide `json:"arrival_guide"`
	Essentials   *Essentials   `json:"essentials"`
}

type Essentials struct {
	Currency string `json:"currency"`
	Category string `json:"category"`
	Vibe     string `json:"vibe"`
}

// PlaceLibraryItem represents a cached place enrichment result
type PlaceLibraryItem struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	GooglePlaceID string    `json:"google_place_id" db:"google_place_id"`
	Description   string    `json:"description" db:"description"`
	Photos        any       `json:"photos" db:"photos"` // Stores JSONB
	Rating        float64   `json:"rating" db:"rating"`
	Category      string    `json:"category" db:"category"`
	Address       string    `json:"address" db:"address"`
	Latitude      float64   `json:"latitude" db:"latitude"`
	Longitude     float64   `json:"longitude" db:"longitude"`
	Website       string    `json:"website" db:"website"`
	Phone         string    `json:"phone" db:"phone"`
	OpeningHours  any       `json:"opening_hours" db:"opening_hours"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type ItineraryResponse struct {
	Itinerary       []ItineraryDay  `json:"itinerary"`
	MorningBriefing string          `json:"morning_briefing"`
	Highlights      []TripHighlight `json:"highlights"`
}

type EditorialResponse struct {
	Tagline           string              `json:"tagline"`
	Vibes             []string            `json:"vibes"`
	MorningBriefing   string              `json:"morning_briefing"`
	HistorySnippet    string              `json:"history_snippet"`
	Highlights        []TripHighlight     `json:"highlights"`
	CulinarySignature []CulinarySignature `json:"culinary_signature"`
	HiddenGem         *HiddenGem          `json:"hidden_gem"`
}

type TripHighlight struct {
	Title       string `json:"title"`
	Type        string `json:"type"` // e.g. "Nature", "Culture"
	Hook        string `json:"hook"` // Hook atau alasan singkat
	ImagePrompt string `json:"image_prompt"`
}

type ArrivalGuide struct {
	PrimaryTransport    string `json:"primary_transport"`     // "Plane"
	TravelTime          string `json:"travel_time"`           // "6h 30m"
	EstimatedPriceRange string `json:"estimated_price_range"` // "$200 - $400"
	VisaInfo            string `json:"visa_info"`
	BestTimeVisit       string `json:"best_time_visit"`
}

type PackingCategory struct {
	Category string   `json:"category"` // "Clothing"
	Items    []string `json:"items"`    // ["Light jacket", "Sneakers"]
}

// ==========================================
// 2. ITINERARY DETAILS
// ==========================================

type ItineraryDay struct {
	Day             int              `json:"day"`
	Title           string           `json:"title"`
	MorningBriefing *MorningBriefing `json:"morning_briefing"`
	Activities      []Activity       `json:"activities"`
}

type MorningBriefing struct {
	WeatherForecast string `json:"weather_forecast"`
	OutfitTip       string `json:"outfit_tip"`
	LocalVibe       string `json:"local_vibe"`
}

type Activity struct {
	Time             string `json:"time"`
	Activity         string `json:"activity"`
	Type             string `json:"type"`
	Description      string `json:"description"`
	DescriptionShort string `json:"description_short"`
	PlaceName        string `json:"place_name"`
	IsSkeleton       bool   `json:"is_skeleton"`

	// Enrichment Fields
	PlaceID  string `json:"place_id,omitempty"`
	Address  string `json:"address,omitempty"`
	ImageURL string `json:"image_url,omitempty"`

	Latitude    *float64     `json:"latitude"`
	Longitude   *float64     `json:"longitude"`
	Coordinates *Coordinates `json:"coordinates"`

	TransitTime   string `json:"transit_time"`
	TransitMethod string `json:"transit_method"`
	TransitPrice  int64  `json:"transit_price"`
	LocationType  string `json:"location_type"`

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
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	PlaceID         string    `json:"place_id"`
	Photos          []string  `json:"photos"`
	VisitDuration   string    `json:"visit_duration"`
	PopularityScore int       `json:"popularity_score"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ==========================================
// 3. OPTIONS & BREAKDOWN (Value Objects)
// ==========================================

type BudgetBreakdown struct {
	Transport     int64 `json:"transport"`
	Accommodation int64 `json:"accommodation"`
	Food          int64 `json:"food"`
	Tickets       int64 `json:"tickets"`
	Misc          int64 `json:"misc"`
}

type TransportBreakdown struct {
	FirstMile string `json:"first_mile"`
	MainLeg   string `json:"main_leg"`
	LastMile  string `json:"last_mile"`
}

type HubDetails struct {
	DepartureNode string `json:"departure_node"`
	ArrivalNode   string `json:"arrival_node"`
}

type LogisticsContext struct {
	DistanceKM   int    `json:"distance_km"`
	RouteType    string `json:"route_type"`
	WarningAlert string `json:"warning_alert"`
}

type TransportOption struct {
	StrategyTag          string             `json:"strategy_tag"`
	Name                 string             `json:"name"`
	PriceTier            string             `json:"price_tier"`
	TotalDurationDisplay string             `json:"total_duration_display"`
	HubDetails           HubDetails         `json:"hub_details"`
	Breakdown            TransportBreakdown `json:"breakdown"`
	OperatorsHint        string             `json:"operators_hint"`
	BookingQuery         string             `json:"booking_query"`
	Pros                 string             `json:"pros"`
}

type TripAndPlan struct {
	Trip Trip     `json:"trip"`
	Plan TripPlan `json:"plan"`
}

// ==========================================
// 4. PACKING & OTHERS
// ==========================================

// 3. Updated Accommodation Option
type AccommodationOption struct {
	Type                 string   `json:"type"`              // Hotel | Villa
	AreaName             string   `json:"area_name"`         // Ganti location_area
	RecommendationReason string   `json:"reason"`            // FIXED: Matches prompt "reason"
	Vibe                 string   `json:"vibe"`              // Ganti description
	HotelSuggestions     []string `json:"hotel_suggestions"` // CHANGED from string to []string
}

// ==========================================
// 4. AI PARSING STRUCTS
// ==========================================

// AIPlannerResponse: Struct bayangan untuk menangkap Raw JSON dari AI Planner
type AIPlannerResponse struct {
	Itinerary            []ItineraryDay        `json:"itinerary"`
	BudgetBreakdown      BudgetBreakdown       `json:"budget_breakdown"`
	TransportOptions     []TransportOption     `json:"transport_options"`
	AccommodationOptions []AccommodationOption `json:"strategic_accommodation"`
	DecisionNotes        []string              `json:"decision_notes"`
	ArrivalGuide         *ArrivalGuide         `json:"arrival_guide"` // New Field
	PackingList          []PackingCategory     `json:"packing_list"`  // New Field
	MorningBriefing      string                `json:"morning_briefing"`
	Highlights           []TripHighlight       `json:"highlights"`
	Logistics            *LogisticsData        `json:"logistics"` // NEW: M-124
	Tagline              string                `json:"tagline"`
	Vibes                []string              `json:"vibes"`
	CulinarySignature    []CulinarySignature   `json:"culinary_signature"`
	HiddenGem            *HiddenGem            `json:"hidden_gem"`
	HistorySnippet       string                `json:"history_snippet"`
}

type TripVibeResponse struct {
	ItineraryUpdates []struct {
		Day           int    `json:"day"`
		ActivityIndex int    `json:"activity_index"`
		Description   string `json:"description"`
		VisitDuration string `json:"visit_duration"`
		Category      string `json:"category"`
	} `json:"itinerary_updates"`
	MorningBriefings []struct {
		Day             int    `json:"day"`
		WeatherForecast string `json:"weather_forecast"`
		OutfitTip       string `json:"outfit_tip"`
		LocalVibe       string `json:"local_vibe"`
	} `json:"morning_briefings"`
	Highlights []TripHighlight `json:"highlights"`
}

type TripLogisticsResponse struct {
	ArrivalGuide           ArrivalGuide          `json:"arrival_guide"`
	BudgetBreakdown        BudgetBreakdown       `json:"budget_breakdown"`
	PackingList            []PackingCategory     `json:"packing_list"`
	StrategicAccommodation []AccommodationOption `json:"strategic_accommodation"`
}

type TripOverviewResponse struct {
	TripTitle              string                `json:"trip_title"`
	MorningBriefing        string                `json:"morning_briefing"`
	ArrivalGuide           ArrivalGuide          `json:"arrival_guide"`
	BudgetBreakdown        BudgetBreakdown       `json:"budget_breakdown"`
	Highlights             []TripHighlight       `json:"highlights"`
	StrategicAccommodation []AccommodationOption `json:"strategic_accommodation"`
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

// ==========================================
// 7. USER & SUBSCRIPTION
// ==========================================

type User struct {
	ID                    int        `json:"id"`
	UserID                string     `json:"user_id"` // Clerk ID
	Email                 string     `json:"email"`
	Name                  string     `json:"name"`
	SubscriptionTier      string     `json:"subscription_tier"`
	SubscriptionStatus    string     `json:"subscription_status"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at"`
	SubscriptionEndsAt    *time.Time `json:"subscription_ends_at"`
	StripeCustomerID      string     `json:"stripe_customer_id"`
	StripeSubscriptionID  string     `json:"stripe_subscription_id"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type TripQuota struct {
	UserID       string    `json:"user_id"`
	Month        string    `json:"month"` // YYYY-MM
	TripsCreated int       `json:"trips_created"`
	QuotaLimit   int       `json:"quota_limit"`
	Remaining    int       `json:"remaining"`    // Virtual field
	IsUnlimited  bool      `json:"is_unlimited"` // Virtual field
	LastReset    time.Time `json:"last_reset"`
}

type SubscriptionEvent struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	EventType     string    `json:"event_type"`
	FromTier      string    `json:"from_tier"`
	ToTier        string    `json:"to_tier"`
	StripeEventID string    `json:"stripe_event_id"`
	Metadata      string    `json:"metadata"` // JSON string
	CreatedAt     time.Time `json:"created_at"`
}
