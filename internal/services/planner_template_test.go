package services_test

import (
	"context"
	"testing"
	"travelmate/internal/domain"
	"travelmate/internal/services"
)

func TestTemplatePlanner_GeneratePlan(t *testing.T) {
	// 1. Setup
	planner := services.NewTemplatePlanner()
	ctx := context.Background()

	// 2. Define Test Cases
	tests := []struct {
		name          string
		input         domain.Trip
		expectedDays  int
		expectedStyle string // Kita cek di decision notes atau itinerary title
	}{
		{
			name: "Relaxed 3 Days Trip",
			input: domain.Trip{
				ID:          "trip-1",
				Destination: "Bali",
				TripDays:    3,
				Style:       "relaxed",
			},
			expectedDays:  3,
			expectedStyle: "relaxed",
		},
		{
			name: "Fast/Cultural 5 Days Trip",
			input: domain.Trip{
				ID:          "trip-2",
				Destination: "Jogja",
				TripDays:    5,
				Style:       "cultural",
			},
			expectedDays:  5,
			expectedStyle: "cultural",
		},
	}

	// 3. Execution
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.GeneratePlan(ctx, tt.input)

			// Assertions
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// Cek apakah ID trip terbawa
			if plan.TripID != tt.input.ID {
				t.Errorf("Expected TripID %s, got %s", tt.input.ID, plan.TripID)
			}

			// Cek jumlah hari itinerary
			if len(plan.Itinerary) != tt.expectedDays {
				t.Errorf("Expected itinerary length %d, got %d", tt.expectedDays, len(plan.Itinerary))
			}

			// Cek basic logic hari pertama
			if plan.Itinerary[0].Day != 1 {
				t.Errorf("Expected first day to be 1, got %d", plan.Itinerary[0].Day)
			}

			// Cek apakah budget breakdown terisi
			if plan.BudgetBreakdown.Transport == 0 {
				t.Error("Expected budget breakdown to be populated")
			}
		})
	}
}
