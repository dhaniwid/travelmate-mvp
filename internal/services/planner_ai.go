package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"travelmate/internal/domain"

	openai "github.com/sashabaranov/go-openai"
)

// ============================================================================
// CONSTANTS & TYPES
// ============================================================================

const (
	ModelFast = openai.GPT4oMini
	ModelRich = openai.GPT4o
)

type AIPlanner struct {
	client    *openai.Client
	promptSvc *PromptService
}

func NewAIPlanner(apiKey string, promptSvc *PromptService) *AIPlanner {
	client := openai.NewClient(apiKey)
	return &AIPlanner{client: client, promptSvc: promptSvc}
}

// ============================================================================
// PUBLIC METHODS (Business Logic Entry Points)
// ============================================================================

// GetDiscoveryInfo Fitur "Dream/Inspiration"
func (p *AIPlanner) GetDiscoveryInfo(ctx context.Context, city string) (*domain.DiscoveryResponse, error) {
	// 1. Siapkan Data untuk Template ({{.Destination}})
	inputData := map[string]string{
		"Destination": city,
	}

	// 2. Variable penampung Raw JSON
	var rawResponse json.RawMessage

	// 3. Panggil AI dengan fungsi generic
	if err := p.requestAI(ctx, "discovery_agent", inputData, &rawResponse); err != nil {
		return nil, fmt.Errorf("discovery AI failed: %w", err)
	}

	// 4. Cleaning & Parsing (Hybrid: Object/Array safe)
	// Karena prompt discovery_agent outputnya Object, kita langsung unmarshal
	cleanData := cleanJSON(rawResponse)

	var resp domain.DiscoveryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("❌ Failed JSON Discovery: %s\n", string(cleanData))
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &resp, nil
}

// GenerateOnlyItinerary Fitur "Plan Itinerary"
func (p *AIPlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error) {
	var rawResponse json.RawMessage

	// 1. Log Start Request
	//fmt.Printf("🔍 [AI DEBUG] Requesting Itinerary for Trip ID: %s, Destination: %s\n", trip.ID, trip.Destination)

	if err := p.requestAI(ctx, "planner_itinerary_system", trip, &rawResponse); err != nil {
		fmt.Printf("❌ [AI DEBUG] Request AI Failed: %v\n", err)
		return nil, err
	}

	cleanData := cleanJSON(rawResponse)

	// --- Percobaan Parsing 1: Wrapper Object ---
	var wrapper struct {
		Itinerary []domain.ItineraryDay `json:"itinerary"`
	}
	if err := json.Unmarshal(cleanData, &wrapper); err == nil && len(wrapper.Itinerary) > 0 {
		return wrapper.Itinerary, nil
	} else if err != nil {
		fmt.Printf("⚠️ [AI DEBUG] Wrapper Unmarshal failed: %v\n", err)
	}

	// --- Percobaan Parsing 2: Direct Array ---
	var directArray []domain.ItineraryDay
	if err := json.Unmarshal(cleanData, &directArray); err == nil && len(directArray) > 0 {
		return directArray, nil
	} else if err != nil {
		fmt.Printf("⚠️ [AI DEBUG] Direct Array Unmarshal failed: %v\n", err)
	}

	fmt.Println("❌ [AI DEBUG] Failed to parse itinerary JSON in both formats")
	return nil, fmt.Errorf("failed to parse itinerary JSON")
}

// GenerateTransportAndStay Fitur "Expert Logistics"
func (p *AIPlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	var logisticsResp struct {
		LogisticsContext       domain.LogisticsContext      `json:"logistics_context"`
		TransportOptions       []domain.TransportOption     `json:"transport_options"`
		StrategicAccommodation []domain.AccommodationOption `json:"strategic_accommodation"`
	}

	// 1. Request ke AI
	if err := p.requestAI(ctx, "planner_logistics_system", trip, &logisticsResp); err != nil {
		fmt.Printf("❌ Logistics AI Error: %v\n", err)
		return domain.TripPlan{}, err
	}

	//fmt.Printf("🛎️ AI Logistics Response: %+v\n", logisticsResp)

	// 2. Return TripPlan (Map dari temporary struct ke domain)
	return domain.TripPlan{
		TripID:               trip.ID,
		LogisticsContext:     logisticsResp.LogisticsContext,
		TransportOptions:     logisticsResp.TransportOptions,
		AccommodationOptions: logisticsResp.StrategicAccommodation,
	}, nil
}

// GenerateAlternatives Fitur "Activity Alternatives"
func (p *AIPlanner) GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	// 1. Siapkan Data untuk Template Prompt
	inputData := map[string]interface{}{
		"Destination": dest,
		"Activity":    activity,
		"Location":    location,
		"Tags":        tags,
	}

	// 2. Request AI
	var alternatives []domain.ActivityAlternative

	if err := p.requestAI(ctx, "planner_alternatives_system", inputData, &alternatives); err != nil {
		return nil, fmt.Errorf("failed to generate alternatives: %w", err)
	}

	return alternatives, nil
}

// GeneratePackingList Membuat daftar bawaan cerdas berdasarkan destinasi, durasi, dan style trip
func (p *AIPlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error) {
	// 1. Siapkan struct wrapper untuk menangkap JSON output
	var result struct {
		PackingList []domain.PackingItem `json:"packing_list"`
	}

	// 2. Request ke OpenAI
	err := p.requestAI(ctx, "planner_packing_system", trip, &result)
	if err != nil {
		log.Printf("❌ Packing List Generation Error: %v", err)
		return nil, err
	}

	return result.PackingList, nil
}

// ============================================================================
// MOCK / FALLBACK LOGIC
// ============================================================================

// generateMockPlan adalah fungsi fallback untuk membuat rencana perjalanan mock jika AI gagal
func (s *AIPlanner) generateMockPlan(req domain.Trip, realTickets []domain.TransportOption) domain.TripPlan {
	log.Println("⚠️ OpenAI Error or JSON Invalid. Switching to Mock Plan.")

	// 1. Dummy Itinerary (Logic sederhana tetap sama)
	itinerary := []domain.ItineraryDay{}
	for i := 1; i <= req.TripDays; i++ {
		day := domain.ItineraryDay{Day: i}
		if i == 1 {
			day.Title = "Arrival & Settlement"
			day.Activities = []domain.Activity{
				{Time: "14:00", Activity: "Check-in Hotel", Type: "Logistics", PlaceName: "Hotel Area", Description: "Check in process"},
				{Time: "19:00", Activity: "Local Dinner", Type: "Culinary", PlaceName: "City Center", Description: "Welcome dinner"},
			}
		} else {
			day.Title = "Exploration Day"
			day.Activities = []domain.Activity{
				{Time: "09:00", Activity: "Main Attraction Visit", Type: "Sightseeing", PlaceName: "Famous Landmark", Description: "Explore the icon"},
				{Time: "13:00", Activity: "Lunch at Local Resto", Type: "Culinary", PlaceName: "Legendary Resto", Description: "Lunch break"},
			}
		}
		itinerary = append(itinerary, day)
	}

	// 2. Dummy Transport (SESUAIKAN DENGAN STRUCT BARU "Expert Logistics")
	transportOpts := []domain.TransportOption{
		{
			StrategyTag:          "CEPAT",
			Name:                 "Direct Flight (Mock)",
			PriceTier:            "HIGH",
			TotalDurationDisplay: "3h 30m", // Field Baru
			Pros:                 "Fastest way to reach destination (Mock Data).",
			OperatorsHint:        "Garuda Indonesia, Citilink",    // Field Baru
			BookingQuery:         "flight jakarta to destination", // Field Baru
			Breakdown: domain.TransportBreakdown{
				FirstMile: "Taxi to Airport (45m)",
				MainLeg:   "Direct Flight (1h)",
				LastMile:  "Taxi to Hotel (30m)",
			},
		},
		{
			StrategyTag:          "HEMAT",
			Name:                 "Intercity Train (Mock)",
			PriceTier:            "LOW",
			TotalDurationDisplay: "9h 00m", // Field Baru
			Pros:                 "Budget friendly option.",
			OperatorsHint:        "KAI (Kereta Api Indonesia)",  // Field Baru
			BookingQuery:         "tiket kereta ke destination", // Field Baru
			Breakdown: domain.TransportBreakdown{
				FirstMile: "Ojek to Station (30m)",
				MainLeg:   "Economy Train (8h)",
				LastMile:  "Angkot to Area (30m)",
			},
		},
	}

	// 3. Dummy Accommodation (SESUAIKAN DENGAN STRUCT BARU)
	accomOpts := []domain.AccommodationOption{
		{
			Type:                 "Hotel",
			AreaName:             "City Center Zone",                   // Ganti LocationArea
			RecommendationReason: "Strategic access to all landmarks.", // Ganti LocationNote
			Vibe:                 "Vibrant area perfect for tourists.", // Ganti Description
		},
		{
			Type:                 "Villa",
			AreaName:             "Quiet Highlands",
			RecommendationReason: "Best for relaxation.",
			Vibe:                 "Peaceful area away from traffic.",
		},
	}

	// 4. Dummy Context
	logContext := domain.LogisticsContext{
		DistanceKM:   123,
		RouteType:    "Inter-City",
		WarningAlert: "⚠️ MOCK PLAN: AI Service Unavailable.",
	}

	// 5. Dummy Budget
	budget := domain.BudgetBreakdown{
		Transport:     2000000,
		Accommodation: 3000000,
		Food:          1500000,
		Tickets:       500000,
		Misc:          500000,
	}

	return domain.TripPlan{
		TripID:               req.ID,
		Itinerary:            itinerary,
		BudgetBreakdown:      budget,
		TransportOptions:     transportOpts,
		AccommodationOptions: accomOpts,
		LogisticsContext:     logContext,
		DecisionNotes:        []string{"⚠️ This is a generated mock plan because AI service is unavailable."},
	}
}

// ---------------------------------------------------------
// PRIVATE HELPER
// ---------------------------------------------------------

// requestAI adalah fungsi TUNGGAL untuk menangani komunikasi ke OpenAI.
func (p *AIPlanner) requestAI(ctx context.Context, sysKey string, data interface{}, target interface{}) error {
	// 1. Render System Prompt (Rules & Schema)
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, data)
	if err != nil {
		return fmt.Errorf("render prompt error [%s]: %w", sysKey, err)
	}

	// 2. Prepare User Content (Data Trip Explicit)
	userDataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}
	userContent := fmt.Sprintf("Here is the trip context data:\n%s", string(userDataBytes))

	// 🔍 DEBUG LOG (Optional)
	fmt.Printf("🤖 [System]: %s\n", sysKey)
	fmt.Printf("👤 [User Context]: %s\n", userContent)

	// 3. Call OpenAI (Multi-role Messages)
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			// Role System: Menjelaskan SIAPA dia dan FORMAT apa yang diminta
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},

			// Role User: Memberikan DATA KONTEKS spesifik
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature:    0.7,
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})

	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("openai returned empty choices")
	}

	// 4. Processing Response
	rawContent := resp.Choices[0].Message.Content
	cleanContent := cleanJSON([]byte(rawContent))

	if err := json.Unmarshal(cleanContent, target); err != nil {
		// Log content jika error syntax, biar mudah debug
		fmt.Printf("❌ JSON Syntax Error for [%s]. \nContent: %s\n", sysKey, cleanContent)
		return fmt.Errorf("json syntax error: %w", err)
	}

	return nil
}

// Helper untuk membersihkan markdown code block (```json ... ```)
func cleanJSON(raw []byte) []byte {
	s := string(raw)
	s = strings.TrimSpace(s)

	// Hapus ```json di awal
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}

	// Hapus ``` di akhir
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return []byte(strings.TrimSpace(s))
}
