package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type EnrichmentService struct {
	TripRepo     *repositories.TripRepository
	GoogleAPIKey string
}

func NewEnrichmentService(tripRepo *repositories.TripRepository, apiKey string) *EnrichmentService {
	return &EnrichmentService{
		TripRepo:     tripRepo,
		GoogleAPIKey: apiKey,
	}
}

// EnrichTrip runs a background job to fetch real photos & coordinates for activities
func (s *EnrichmentService) EnrichTrip(ctx context.Context, tripID string) {
	log.Printf("✨ [Enrichment] Starting for Trip ID: %s", tripID)

	// 1. Fetch Trip WITH Plan
	tripAndPlan, err := s.TripRepo.GetTripWithPlan(ctx, tripID)
	if err != nil {
		log.Printf("❌ [Enrichment] Failed to get trip: %v", err)
		return
	}
	if tripAndPlan == nil {
		log.Printf("❌ [Enrichment] Trip %s not found", tripID)
		return
	}

	// Reference to Plan
	plan := &tripAndPlan.Plan
	trip := &tripAndPlan.Trip

	if len(plan.Itinerary) == 0 {
		log.Printf("⚠️ [Enrichment] No itinerary to enrich for trip %s", tripID)
		return
	}

	// 2. Iterate & Enrich (Limit concurrency to avoid rate limits)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Max 5 requests at a time

	enrichedCount := 0
	var mu sync.Mutex

	// Time generation: Start at 09:00 for each day
	for dayIdx, day := range plan.Itinerary {
		currentTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC) // 09:00 AM start
		for actIdx, activity := range day.Activities {
			// Skip if already enriched or generic
			if activity.PlaceID != "" || activity.Type == "Logistics" {
				continue
			}

			wg.Add(1)
			go func(dIdx, aIdx int, act domain.Activity, dest string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				// Query: Use activity name (e.g., "Explore Shinjuku in Tokyo")
				// Fallback to place_name if activity is empty
				queryName := act.Activity
				if queryName == "" && act.PlaceName != "" {
					queryName = act.PlaceName
				}
				query := fmt.Sprintf("%s in %s", queryName, dest)
				place, err := s.findPlace(query)
				if err == nil && place != nil {
					mu.Lock()
					// Update Activity in memory
					plan.Itinerary[dIdx].Activities[aIdx].PlaceID = place.PlaceID
					plan.Itinerary[dIdx].Activities[aIdx].Address = place.FormattedAddress

					// NEW: Replace generic activity name with real POI name
					if place.Name != "" {
						plan.Itinerary[dIdx].Activities[aIdx].Activity = place.Name
						plan.Itinerary[dIdx].Activities[aIdx].PlaceName = place.Name
					}

					// NEW: Classify activity type from Google Places types
					activityType := s.classifyActivityType(place.Types)
					plan.Itinerary[dIdx].Activities[aIdx].Type = activityType

					if place.Geometry.Location.Lat != 0 {
						valLat := place.Geometry.Location.Lat
						valLng := place.Geometry.Location.Lng
						// Update Pointers to float64
						plan.Itinerary[dIdx].Activities[aIdx].Latitude = &valLat
						plan.Itinerary[dIdx].Activities[aIdx].Longitude = &valLng
						// Update legacy Coordinates struct
						plan.Itinerary[dIdx].Activities[aIdx].Coordinates = &domain.Coordinates{
							Lat: valLat,
							Lng: valLng,
						}
					}

					// Photo (ambil reference pertama)
					if len(place.Photos) > 0 {
						// Google Places Photo URL format
						// https://maps.googleapis.com/maps/api/place/photo?maxwidth=400&photo_reference=...&key=...
						// Kita simpan FULL URL agar frontend tinggal render
						photoRef := place.Photos[0].PhotoReference
						photoURL := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photo_reference=%s&key=%s",
							photoRef, s.GoogleAPIKey)

						plan.Itinerary[dIdx].Activities[aIdx].ImageURL = photoURL
					}
					enrichedCount++
					mu.Unlock()
				}
			}(dayIdx, actIdx, activity, trip.Destination)
		}

		// NEW: Assign times to activities after enrichment
		for aIdx := range day.Activities {
			plan.Itinerary[dayIdx].Activities[aIdx].Time = currentTime.Format("15:04")

			// Increment time for next activity
			// Meals get 2 hours, other activities get 3 hours
			actType := plan.Itinerary[dayIdx].Activities[aIdx].Type
			if strings.Contains(strings.ToLower(actType), "culinary") || strings.Contains(strings.ToLower(actType), "food") {
				currentTime = currentTime.Add(time.Hour * 2)
			} else {
				currentTime = currentTime.Add(time.Hour * 3)
			}

			// Don't schedule activities past 21:00
			if currentTime.Hour() > 21 {
				break
			}
		}
	}

	wg.Wait()

	// 3. Save Context Lat/Long (from first activity or destination)
	// (Optional: Set Trip Location based on result)

	// 4. Save Updates to DB
	// 4. Save Updates to DB
	// We always try to save because time generation and type defaults run regardless of API success
	if err := s.TripRepo.SaveTripPlan(ctx, *trip, *plan); err != nil {
		log.Printf("❌ [Enrichment] Failed to save updates: %v", err)
	} else {
		if enrichedCount > 0 {
			log.Printf("✅ [Enrichment] Successfully enriched %d activities (API) + structured data for %s", enrichedCount, trip.Destination)
		} else {
			log.Printf("✅ [Enrichment] Added structured data (time/types) for %s (No external API matches)", trip.Destination)
		}
	}
}

// Internal Structs for Google Places API
type GooglePlacesSearchResponse struct {
	Candidates []GooglePlacesSearchResponseCandidate `json:"candidates"`
	Status     string                                `json:"status"`
}

func (s *EnrichmentService) findPlace(query string) (*GooglePlacesSearchResponseCandidate, error) {
	if s.GoogleAPIKey == "" {
		return nil, fmt.Errorf("no api key")
	}

	endpoint := "https://maps.googleapis.com/maps/api/place/findplacefromtext/json"
	params := url.Values{}
	params.Add("input", query)
	params.Add("inputtype", "textquery")
	params.Add("fields", "place_id,name,formatted_address,geometry,photos,types") // Added 'types' for classification
	params.Add("key", s.GoogleAPIKey)

	resp, err := http.Get(endpoint + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api status: %d", resp.StatusCode)
	}

	var result GooglePlacesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "OK" || len(result.Candidates) == 0 {
		return nil, fmt.Errorf("api status: %s", result.Status)
	}

	return &result.Candidates[0], nil
}

type GooglePlacesSearchResponseCandidate struct {
	PlaceID          string `json:"place_id"`
	FormattedAddress string `json:"formatted_address"`
	Name             string `json:"name"`
	Geometry         struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
	Photos []struct {
		PhotoReference string `json:"photo_reference"`
	} `json:"photos"`
	Types []string `json:"types"` // NEW: For type classification
}

// classifyActivityType maps Google Places types to our activity categories
func (s *EnrichmentService) classifyActivityType(types []string) string {
	// Priority-based classification (first match wins)
	for _, placeType := range types {
		switch placeType {
		// Culinary
		case "restaurant", "cafe", "bar", "food", "meal_takeaway", "bakery":
			return "Culinary"
		// Nature
		case "park", "natural_feature", "campground", "zoo":
			return "Nature"
		// Shopping
		case "shopping_mall", "store", "clothing_store", "jewelry_store", "book_store":
			return "Shopping"
		// Entertainment
		case "night_club", "movie_theater", "casino", "bowling_alley", "amusement_park":
			return "Entertainment"
		// Sightseeing (cultural/historical)
		case "museum", "art_gallery", "church", "synagogue", "mosque", "hindu_temple",
			"tourist_attraction", "landmark", "point_of_interest", "premise":
			return "Sightseeing"
		}
	}

	// Default fallback
	return "Sightseeing"
}
