package services

import (
	"context"
	"strings"
	"testing"
	"travelmate/internal/domain"
)

func TestGetRegeneratePrompt_Preferences(t *testing.T) {
	// 1. Setup Mock Dependencies
	// PromptService with pre-populated cache to avoid DB hit
	promptSvc := NewPromptService(nil)
	promptSvc.addToCache("planner_itinerary_system", "Rules: {{.Constraints}}")

	planner := NewAIPlanner("", promptSvc, nil, nil)

	// 2. Define Test Data
	mockTrip := domain.Trip{
		Destination: "Tokyo",
		TripDays:    5,
	}

	prefs := domain.UserPreferences{
		Dietary: []string{"Vegan"},
		Pace:    "FAST",
	}

	// 3. Execute
	prompt, err := planner.GetRegeneratePrompt(context.Background(), mockTrip, prefs)
	if err != nil {
		t.Fatalf("❌ GetRegeneratePrompt failed: %v", err)
	}

	// 4. Assertions
	t.Logf("Generated Prompt Content:\n%s", prompt)

	if !strings.Contains(prompt, "Vegan") {
		t.Errorf("FAIL: Prompt should contain 'Vegan' dietary restriction")
	}

	if !strings.Contains(prompt, "High Intensity") {
		t.Errorf("FAIL: Prompt should contain 'High Intensity' for FAST pace")
	}

	if !strings.Contains(prompt, "Tokyo") {
		t.Errorf("FAIL: Prompt should contain destination 'Tokyo'")
	}
}
