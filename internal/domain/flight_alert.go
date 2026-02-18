package domain

import "time"

// FlightPriceAlert represents a flight price monitoring entry for a trip
type FlightPriceAlert struct {
	ID                 string     `json:"id" db:"id"`
	TripID             string     `json:"trip_id" db:"trip_id"`
	OriginAirport      string     `json:"origin_airport" db:"origin_airport"`
	DestinationAirport string     `json:"destination_airport" db:"destination_airport"`
	DepartureDate      time.Time  `json:"departure_date" db:"departure_date"`
	ReturnDate         *time.Time `json:"return_date,omitempty" db:"return_date"`
	InitialPrice       float64    `json:"initial_price" db:"initial_price"`
	CurrentPrice       float64    `json:"current_price" db:"current_price"`
	LowestPriceSeen    float64    `json:"lowest_price_seen" db:"lowest_price_seen"`
	Currency           string     `json:"currency" db:"currency"`
	LastCheckedAt      *time.Time `json:"last_checked_at,omitempty" db:"last_checked_at"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// FlightGuardianStatus represents the activation status for UI
type FlightGuardianStatus struct {
	Status       string  `json:"status"` // "active", "inactive", "error"
	CurrentPrice float64 `json:"current_price"`
	Currency     string  `json:"currency"`
	AlertID      string  `json:"alert_id,omitempty"`
}
