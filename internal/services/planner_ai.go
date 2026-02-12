package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"

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
	prefRepo  *repositories.PreferencesRepository // Concrete type
}

func NewAIPlanner(apiKey string, promptSvc *PromptService, prefRepo *repositories.PreferencesRepository) *AIPlanner {
	client := openai.NewClient(apiKey)
	return &AIPlanner{client: client, promptSvc: promptSvc, prefRepo: prefRepo}
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
func (p *AIPlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	var rawResponse json.RawMessage

	// Performance monitoring
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Itinerary AI Request for %s", trip.Destination)

	// Data for Template
	// FIX: Template expects {{.Trip.TripDays}}, so we must wrap it.
	inputData := map[string]interface{}{
		"Trip": trip,
	}

	if err := p.requestAI(ctx, "planner_itinerary_system", inputData, &rawResponse); err != nil {
		log.Printf("❌ [AI ERROR] Request AI Failed: %v", err)
		return domain.ItineraryResponse{}, err
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️ [PERF] Itinerary AI Request completed in: %v", elapsed)

	cleanData := cleanJSON(rawResponse)

	// Define strict schema locally to capture everything
	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("⚠️ [AI DEBUG] Itinerary Unmarshal failed: %v\n", err)
		// Fallback check if it is direct array (old format)
		var directArray []domain.ItineraryDay
		if err2 := json.Unmarshal(cleanData, &directArray); err2 == nil {
			return domain.ItineraryResponse{Itinerary: directArray}, nil
		}
		return domain.ItineraryResponse{}, fmt.Errorf("failed to parse itinerary JSON")
	}

	return resp, nil
}

// GenerateEditorial Fitur "Magazine Editor" (Phase 2 Parallel)
func (p *AIPlanner) GenerateEditorial(ctx context.Context, trip domain.Trip) (domain.EditorialResponse, error) {
	var rawResponse json.RawMessage

	// Performance monitoring
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Editorial AI Request for %s", trip.Destination)

	if err := p.requestAI(ctx, "planner_editorial_system", trip, &rawResponse); err != nil {
		log.Printf("❌ [AI ERROR] Editorial AI Failed: %v", err)
		return domain.EditorialResponse{}, err
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️ [PERF] Editorial AI Request completed in: %v", elapsed)

	cleanData := cleanJSON(rawResponse)
	var resp domain.EditorialResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("⚠️ [AI DEBUG] Editorial Unmarshal failed: %v\n", err)
		return domain.EditorialResponse{}, err
	}

	return resp, nil
}

// GenerateTransportAndStay Fitur "Expert Logistics"
func (p *AIPlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	var logisticsResp struct {
		ArrivalGuide           domain.ArrivalGuide          `json:"arrival_guide"`
		StrategicAccommodation []domain.AccommodationOption `json:"strategic_accommodation"`
		BudgetBreakdown        domain.BudgetBreakdown       `json:"budget_breakdown"`
	}

	// 1. Request ke AI - Use specialized logistics prompt
	if err := p.requestAI(ctx, "planner_logistics_system", trip, &logisticsResp); err != nil {
		log.Printf("❌ Logistics AI Error: %v", err)
		return domain.TripPlan{}, err
	}

	// 2. Return TripPlan (Map dari temporary struct ke domain)
	return domain.TripPlan{
		TripID:               trip.ID,
		ArrivalGuide:         &logisticsResp.ArrivalGuide,
		AccommodationOptions: logisticsResp.StrategicAccommodation,
		BudgetBreakdown:      logisticsResp.BudgetBreakdown,
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
func (p *AIPlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingCategory, error) {
	// 1. Siapkan struct wrapper untuk menangkap JSON output
	var result struct {
		PackingList []domain.PackingCategory `json:"packing_list"`
	}

	// 2. Request ke OpenAI
	err := p.requestAI(ctx, "planner_packing_system", trip, &result)
	if err != nil {
		log.Printf("❌ Packing List Generation Error: %v", err)
		return nil, err
	}

	return result.PackingList, nil
}

// GeneratePlan Menggabungkan Itinerary dan Logistik ke dalam satu rencana perjalanan penuh
func (p *AIPlanner) GeneratePlan(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	// NEW: Monolithic Prompt Approach (One-Shot Generation)
	// We use "planner_itinerary_system" which now returns EVERYTHING.

	// 1. Fetch User Preferences (Travel DNA)
	prefs, err := p.prefRepo.GetPreferences(ctx, trip.UserID)
	if err != nil {
		log.Printf("⚠️ Failed to fetch preferences for user %s: %v", trip.UserID, err)
		// Continue without preferences
	}

	// 2. Prepare Context (Trip + DNA)
	tripContext := map[string]interface{}{
		"Trip":        trip,
		"Preferences": prefs, // Can be nil, which is fine
	}

	var rawResponse json.RawMessage
	if err := p.requestAI(ctx, "planner_itinerary_system", tripContext, &rawResponse); err != nil {
		fmt.Printf("❌ [AI DEBUG] Request AI Failed: %v\n", err)
		return p.generateMockPlan(trip, nil), nil // Fallback
	}

	cleanData := cleanJSON(rawResponse)

	// Define strict schema locally to capture everything
	var fullResponse domain.AIPlannerResponse
	if err := json.Unmarshal(cleanData, &fullResponse); err != nil {
		fmt.Printf("❌ [AI DEBUG] Full Plan Unmarshal Failed: %v\n", err)
		// Try to recover just itinerary if possible, or fallback
		return p.generateMockPlan(trip, nil), nil
	}

	// Map to Domain TripPlan
	plan := domain.TripPlan{
		TripID:               trip.ID,
		Itinerary:            fullResponse.Itinerary,
		BudgetBreakdown:      fullResponse.BudgetBreakdown,
		TransportOptions:     fullResponse.TransportOptions,
		AccommodationOptions: fullResponse.AccommodationOptions,
		DecisionNotes:        fullResponse.DecisionNotes,
		ArrivalGuide:         fullResponse.ArrivalGuide,
		PackingList:          fullResponse.PackingList,
		MorningBriefing:      fullResponse.MorningBriefing,
		Highlights:           fullResponse.Highlights,
		Tagline:              fullResponse.Tagline,
		Vibes:                fullResponse.Vibes,
		CulinarySignature:    fullResponse.CulinarySignature,
		HiddenGem:            fullResponse.HiddenGem,
		HistorySnippet:       fullResponse.HistorySnippet,
	}

	// If parts are missing, we might want to fill them with dummy data or leave empty
	// For now, return what we got.

	return plan, nil
}

// RefineItinerary Modifies an existing itinerary based on user instruction (Chat Agent)
func (p *AIPlanner) RefineItinerary(ctx context.Context, currentItinerary []domain.ItineraryDay, instruction string) ([]domain.ItineraryDay, error) {
	// 1. Prepare Context Data
	inputData := map[string]interface{}{
		"CurrentItinerary": currentItinerary,
		"Instruction":      instruction,
	}

	// 2. Struct to capture response
	var response struct {
		Itinerary []domain.ItineraryDay `json:"itinerary"`
	}

	// 3. Request AI
	if err := p.requestAI(ctx, "planner_refinement_system", inputData, &response); err != nil {
		fmt.Printf("❌ Refinement AI Failed: %v\n", err)
		return nil, err
	}

	return response.Itinerary, nil
}

// ============================================================================
// MOCK / FALLBACK LOGIC
// ============================================================================

// generateMockPlan adalah fungsi fallback untuk membuat rencana perjalanan mock jika AI gagal
func (s *AIPlanner) generateMockPlan(req domain.Trip, _ []domain.TransportOption) domain.TripPlan {
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
		LogisticsContext:     &logContext,
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

	// 🔍 PERFORMANCE DEBUG
	// log.Printf("📊 [AI-PAYLOAD] System: %s | UserContent Size: %d bytes", sysKey, len(userContent))

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

	// Step 1: Log the Raw AI Response for Debugging
	// log.Printf("📥 [AI-RESPONSE] System: %s | Length: %d chars", sysKey, len(cleanContent))

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
	s = strings.TrimSuffix(s, "```")

	return []byte(strings.TrimSpace(s))
}

// GenerateUltraConciseItinerary creates the initial skeleton (Phase 1)
func (p *AIPlanner) GenerateUltraConciseItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	// Performance monitoring
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Concise Itinerary AI Request (Phase 1) for %s...", trip.Destination)

	var rawResponse json.RawMessage

	// Data for Template
	inputData := map[string]interface{}{
		"Trip": trip,
	}

	// Use the new "Ultra-Concise" prompt
	if err := p.requestAI(ctx, "planner_itinerary_concise", inputData, &rawResponse); err != nil {
		log.Printf("❌ [AI ERROR] Concise Itinerary Failed: %v", err)
		return domain.ItineraryResponse{}, err
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️ [PERF] Concise Itinerary AI Request completed in: %v", elapsed)

	cleanData := cleanJSON(rawResponse)
	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("concise parse error: %w", err)
	}

	return resp, nil
}

// GenerateFullItineraryPass creates the complete trip plan in one pass
func (p *AIPlanner) GenerateFullItineraryPass(ctx context.Context, trip domain.Trip) (domain.AIPlannerResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Heavy Full-Pass Itinerary AI Request for %s...", trip.Destination)

	var rawResponse json.RawMessage

	inputData := map[string]interface{}{
		"Trip":        trip,
		"TripID":      trip.ID,
		"TripDays":    trip.TripDays,
		"Destination": trip.Destination,
		"Pace":        trip.Style, // Assuming Style as Pace for now, or use trip.Preferences
		"Travelers":   "Solo",     // Defaults
		"Budget":      "Medium",
	}

	if err := p.requestAI(ctx, "planner_itinerary_concise", inputData, &rawResponse); err != nil {
		log.Printf("❌ [AI ERROR] Full-Pass Itinerary Failed: %v", err)
		return domain.AIPlannerResponse{}, err
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️ [PERF] Full-Pass Itinerary AI Request completed in: %v", elapsed)

	cleanData := cleanJSON(rawResponse)
	var resp domain.AIPlannerResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("❌ JSON Unmarshal Error in Full-Pass: %v\nContent: %s\n", err, string(cleanData))
		return domain.AIPlannerResponse{}, fmt.Errorf("full-pass parse error: %w", err)
	}

	return resp, nil
}

// GenerateEnrichmentDetails fleshes out the skeleton (Phase 2)
func (p *AIPlanner) GenerateEnrichmentDetails(ctx context.Context, skeleton domain.TripPlan) (domain.TripPlan, error) {
	// Performance monitoring
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Enrichment AI Request (Phase 2)...")

	var rawResponse json.RawMessage

	// Serialize skeleton for prompt context
	skeletonBytes, _ := json.Marshal(skeleton)
	inputData := map[string]interface{}{
		"SkeletonJSON": string(skeletonBytes),
	}

	if err := p.requestAI(ctx, "planner_enrichment", inputData, &rawResponse); err != nil {
		log.Printf("❌ [AI ERROR] Enrichment Failed: %v", err)
		return domain.TripPlan{}, err
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️ [PERF] Enrichment AI Request completed in: %v", elapsed)

	cleanData := cleanJSON(rawResponse)
	var resp domain.TripPlan
	// Fix: The prompt returns "TripPlan" structure directly (or logically should)
	// We unmarshal into TripPlan
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		// Fallback: maybe it wrapped it in { "plan": ... } ?
		// For now assume prompt follows instruction to return TripPlan structure
		return domain.TripPlan{}, fmt.Errorf("enrichment parse error: %w", err)
	}

	return resp, nil
}

// GenerateTripCore (Stage 1): High-speed core itinerary & coordinates
func (p *AIPlanner) GenerateTripCore(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_CORE AI Request (Stage 1) for %s...", trip.Destination)

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Destination": trip.Destination,
		"TripDays":    trip.TripDays,
		"Pace":        trip.Style,
		"Travelers":   "Solo",
		"Budget":      "Medium",
		"TripID":      trip.ID,
	}

	if err := p.requestAI(ctx, "TRIP_CORE", inputData, &rawResponse); err != nil {
		return domain.ItineraryResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_CORE Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("core parse error: %w", err)
	}

	return resp, nil
}

// EnrichTripVibe (Stage 2): Adds descriptions, briefings, and highlights
func (p *AIPlanner) EnrichTripVibe(ctx context.Context, stage1JSON string) (domain.TripVibeResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_ENRICHMENT AI Request (Stage 2)...")

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Stage1JSON": stage1JSON,
	}

	if err := p.requestAI(ctx, "TRIP_ENRICHMENT", inputData, &rawResponse); err != nil {
		return domain.TripVibeResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_ENRICHMENT Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.TripVibeResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.TripVibeResponse{}, fmt.Errorf("vibe parse error: %w", err)
	}

	return resp, nil
}

// GenerateTripLogistics (Stage 3): Strategic details (Visa, Transport, Accommo, Budget)
func (p *AIPlanner) GenerateTripLogistics(ctx context.Context, trip domain.Trip) (domain.TripLogisticsResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_LOGISTICS AI Request (Stage 3) for %s...", trip.Destination)

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Destination": trip.Destination,
	}

	if err := p.requestAI(ctx, "TRIP_LOGISTICS", inputData, &rawResponse); err != nil {
		return domain.TripLogisticsResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_LOGISTICS Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.TripLogisticsResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.TripLogisticsResponse{}, fmt.Errorf("logistics parse error: %w", err)
	}

	return resp, nil
}
