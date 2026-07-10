package services

import (
	"fmt"
	"strings"
	"travelmate/internal/domain"
)

// validateItineraryResponse checks that a skeleton/itinerary response has the
// minimum required structure. An empty itinerary is the most dangerous failure
// mode — it silently renders a blank trip to the user.
func validateItineraryResponse(resp domain.ItineraryResponse, source string) error {
	if len(resp.Itinerary) == 0 {
		return fmt.Errorf("[%s] AI returned empty itinerary (0 days)", source)
	}
	for i, day := range resp.Itinerary {
		if len(day.Activities) == 0 {
			return fmt.Errorf("[%s] Day %d has 0 activities", source, i+1)
		}
		for j, act := range day.Activities {
			if strings.TrimSpace(act.Activity) == "" {
				return fmt.Errorf("[%s] Day %d, activity %d has empty title", source, i+1, j+1)
			}
		}
	}
	return nil
}

// validateEditorialResponse checks that the editorial block has at least a tagline.
// All other fields are optional — empty highlights or culinary are tolerable.
func validateEditorialResponse(resp domain.EditorialResponse, source string) error {
	if strings.TrimSpace(resp.Tagline) == "" {
		return fmt.Errorf("[%s] AI returned empty tagline in editorial response", source)
	}
	return nil
}

// validateLogisticsResponse checks that logistics has at least one accommodation
// option OR a non-empty arrival guide — at least one meaningful field must be present.
func validateLogisticsResponse(resp domain.TripLogisticsResponse, source string) error {
	hasAccommodation := len(resp.StrategicAccommodation) > 0
	hasArrivalGuide := strings.TrimSpace(resp.ArrivalGuide.PrimaryTransport) != ""
	if !hasAccommodation && !hasArrivalGuide {
		return fmt.Errorf("[%s] AI returned empty logistics (no accommodation, no arrival guide)", source)
	}
	return nil
}

// validateAIPlannerResponse checks the monolithic full-plan response.
func validateAIPlannerResponse(resp domain.AIPlannerResponse, source string) error {
	if len(resp.Itinerary) == 0 {
		return fmt.Errorf("[%s] AI returned empty itinerary in full plan response", source)
	}
	return nil
}

// validateActivityAlternatives checks that the AI returned at least one usable alternative.
func validateActivityAlternatives(alts []domain.ActivityAlternative, source string) error {
	if len(alts) == 0 {
		return fmt.Errorf("[%s] AI returned 0 activity alternatives", source)
	}
	for i, a := range alts {
		if strings.TrimSpace(a.Activity) == "" {
			return fmt.Errorf("[%s] alternative %d has empty activity title", source, i+1)
		}
	}
	return nil
}
