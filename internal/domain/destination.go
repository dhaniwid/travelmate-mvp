package domain

import (
	"encoding/json"
	"time"
)

type Destination struct {
	ID              string          `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Country         string          `json:"country" db:"country"`
	Description     string          `json:"description" db:"description"`
	ImageURL        string          `json:"image_url" db:"image_url"`
	Category        string          `json:"category" db:"category"` // 'Nature', 'City', 'Beach', 'Culinary'
	Tags            json.RawMessage `json:"tags" db:"tags"`         // JSONB in DB
	IsTrending      bool            `json:"is_trending" db:"is_trending"`
	PopularityScore int             `json:"popularity_score" db:"popularity_score"`
	DiscoveryData   json.RawMessage `json:"discovery_data" db:"discovery_data"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}
