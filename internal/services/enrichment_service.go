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
	TripRepo         *repositories.TripRepository
	PlaceLibraryRepo *repositories.PlaceLibraryRepository
	GoogleAPIKey     string
}

func NewEnrichmentService(tripRepo *repositories.TripRepository, placeLibraryRepo *repositories.PlaceLibraryRepository, apiKey string) *EnrichmentService {
	return &EnrichmentService{
		TripRepo:         tripRepo,
		PlaceLibraryRepo: placeLibraryRepo,
		GoogleAPIKey:     apiKey,
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

				// 1. Check local Cache (PlaceLibrary)
				var placeID, address, name, imageURL string
				var lat, lng float64
				var types []string

				cached, _ := s.PlaceLibraryRepo.GetByName(context.Background(), queryName)
				if cached != nil {
					log.Printf("🎯 [Enrichment] Cache HIT for: %s", queryName)
					placeID = cached.GooglePlaceID
					address = cached.Address
					name = cached.Name
					lat = cached.Latitude
					lng = cached.Longitude
					if cached.Category != "" {
						types = []string{cached.Category}
					}
					// Handle Photos from cache
					if photos, ok := cached.Photos.([]interface{}); ok && len(photos) > 0 {
						if photo, ok := photos[0].(map[string]interface{}); ok {
							if url, ok := photo["url"].(string); ok {
								imageURL = url
							}
						}
					}
				} else {
					// 2. Fallback to Google API
					place, err := s.findPlace(query)
					if err == nil && place != nil {
						placeID = place.PlaceID
						address = place.FormattedAddress
						name = place.Name
						lat = place.Geometry.Location.Lat
						lng = place.Geometry.Location.Lng
						types = place.Types

						if len(place.Photos) > 0 {
							photoRef := place.Photos[0].PhotoReference
							imageURL = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photo_reference=%s&key=%s",
								photoRef, s.GoogleAPIKey)
						}

						// Save to local Cache for future
						_ = s.PlaceLibraryRepo.Upsert(context.Background(), &domain.PlaceLibraryItem{
							Name:          queryName,
							GooglePlaceID: placeID,
							Address:       address,
							Latitude:      lat,
							Longitude:     lng,
							Category:      s.classifyActivityType(types),
							Photos:        []map[string]string{{"url": imageURL}},
						})
					}
				}

				if placeID != "" {
					mu.Lock()
					plan.Itinerary[dIdx].Activities[aIdx].PlaceID = placeID
					plan.Itinerary[dIdx].Activities[aIdx].Address = address

					if name != "" {
						plan.Itinerary[dIdx].Activities[aIdx].Activity = name
						plan.Itinerary[dIdx].Activities[aIdx].PlaceName = name
					}

					activityType := s.classifyActivityType(types)
					plan.Itinerary[dIdx].Activities[aIdx].Type = activityType

					if lat != 0 {
						plan.Itinerary[dIdx].Activities[aIdx].Latitude = &lat
						plan.Itinerary[dIdx].Activities[aIdx].Longitude = &lng
						plan.Itinerary[dIdx].Activities[aIdx].Coordinates = &domain.Coordinates{
							Lat: lat,
							Lng: lng,
						}
					}

					if imageURL != "" {
						plan.Itinerary[dIdx].Activities[aIdx].ImageURL = imageURL
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

	// NEW: Sanitize Itinerary (Remove logical hallucinations)
	s.sanitizeItinerary(plan)

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

// EnrichSingleActivity (M-126): Targeted enrichment for lazy loading
func (s *EnrichmentService) EnrichSingleActivity(ctx context.Context, tripID string, dayIdx, actIdx int) (*domain.Activity, error) {
	log.Printf("✨ [Lazy Enrichment] Day %d, Activity %d for Trip %s", dayIdx, actIdx, tripID)

	// 1. Fetch Trip WITH Plan
	tripAndPlan, err := s.TripRepo.GetTripWithPlan(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("trip not found: %w", err)
	}
	plan := &tripAndPlan.Plan
	trip := &tripAndPlan.Trip

	// 2. Locate Activity
	if dayIdx < 0 || dayIdx >= len(plan.Itinerary) {
		return nil, fmt.Errorf("invalid day index: %d", dayIdx)
	}
	day := &plan.Itinerary[dayIdx]
	if actIdx < 0 || actIdx >= len(day.Activities) {
		return nil, fmt.Errorf("invalid activity index: %d", actIdx)
	}
	activity := &day.Activities[actIdx]

	// 3. Enrich if needed
	queryName := activity.PlaceName
	if queryName == "" {
		queryName = activity.Activity
	}
	query := fmt.Sprintf("%s in %s", queryName, trip.Destination)

	var placeID, address, name, imageURL string
	var lat, lng float64
	var types []string

	// Check local Cache First
	cached, _ := s.PlaceLibraryRepo.GetByName(ctx, queryName)
	if cached != nil {
		log.Printf("🎯 [Lazy Enrichment] Cache HIT for: %s", queryName)
		placeID = cached.GooglePlaceID
		address = cached.Address
		name = cached.Name
		lat = cached.Latitude
		lng = cached.Longitude
		if cached.Category != "" {
			types = []string{cached.Category}
		}
		if photos, ok := cached.Photos.([]interface{}); ok && len(photos) > 0 {
			if photo, ok := photos[0].(map[string]interface{}); ok {
				if url, ok := photo["url"].(string); ok {
					imageURL = url
				}
			}
		}
	} else {
		place, err := s.findPlace(query)
		if err == nil && place != nil {
			placeID = place.PlaceID
			address = place.FormattedAddress
			name = place.Name
			lat = place.Geometry.Location.Lat
			lng = place.Geometry.Location.Lng
			types = place.Types

			if len(place.Photos) > 0 {
				photoRef := place.Photos[0].PhotoReference
				imageURL = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photo_reference=%s&key=%s",
					photoRef, s.GoogleAPIKey)
			}

			// Save to local Cache
			_ = s.PlaceLibraryRepo.Upsert(ctx, &domain.PlaceLibraryItem{
				Name:          queryName,
				GooglePlaceID: placeID,
				Address:       address,
				Latitude:      lat,
				Longitude:     lng,
				Category:      s.classifyActivityType(types),
				Photos:        []map[string]string{{"url": imageURL}},
			})
		}
	}

	// 4. Map Results
	if placeID != "" {
		activity.PlaceID = placeID
		activity.Address = address
		activity.IsSkeleton = false

		if name != "" {
			activity.PlaceName = name
			activity.Activity = name
		}

		activityType := s.classifyActivityType(types)
		activity.Type = activityType

		if lat != 0 {
			activity.Latitude = &lat
			activity.Longitude = &lng
			activity.Coordinates = &domain.Coordinates{
				Lat: lat,
				Lng: lng,
			}
		}

		if imageURL != "" {
			activity.ImageURL = imageURL
		}

		// 5. Save back to DB
		if err := s.TripRepo.SaveTripPlan(ctx, *trip, *plan); err != nil {
			log.Printf("❌ [Lazy Enrichment] Failed to save: %v", err)
		}
	}

	return activity, nil
}

// sanitizeItinerary removes logical inconsistencies like "Return to Hotel" after "Departure"
func (s *EnrichmentService) sanitizeItinerary(plan *domain.TripPlan) {
	for i, day := range plan.Itinerary {
		var validActivities []domain.Activity
		hasDeparted := false

		for _, act := range day.Activities {
			// Check if we have already departed
			if hasDeparted {
				// If we have departed, skip any "Return to Hotel" or logistics that imply staying
				// allowed: maybe "Arrive at Airport"? but usually Departure is the last step.
				// We definitely skip "Return to Hotel"
				if strings.Contains(strings.ToLower(act.Activity), "return to hotel") ||
					strings.Contains(strings.ToLower(act.Activity), "back to hotel") {
					continue
				}
				// If it's another activity, we might want to keep it?
				// The hallucination was specifically "Return to Hotel" at 19:00 after 18:00 Departure.
				// Let's be aggressive: if it's "Return to Hotel", skip it.
			}

			// Add activity to valid list
			validActivities = append(validActivities, act)

			// Check if this activity IS a departure
			// Keywords: "Departure", "Flight to", "Train to", "Return to [City]" (but not Hotel)
			lowerName := strings.ToLower(act.Activity)
			if (strings.Contains(lowerName, "departure") ||
				strings.Contains(lowerName, "flight to") ||
				strings.Contains(lowerName, "train to")) &&
				!strings.Contains(lowerName, "hotel") {
				hasDeparted = true
			}
		}

		// Update the day's activities
		plan.Itinerary[i].Activities = validActivities
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
