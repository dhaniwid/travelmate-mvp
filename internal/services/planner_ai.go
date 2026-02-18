package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
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
	client     *openai.Client
	promptSvc  *PromptService
	prefRepo   *repositories.PreferencesRepository
	amadeusSvc *AmadeusService // NEW
}

func NewAIPlanner(apiKey string, promptSvc *PromptService, prefRepo *repositories.PreferencesRepository, amadeusSvc *AmadeusService) *AIPlanner {
	client := openai.NewClient(apiKey)
	return &AIPlanner{client: client, promptSvc: promptSvc, prefRepo: prefRepo, amadeusSvc: amadeusSvc}
}

// ============================================================================
// PUBLIC METHODS (Business Logic Entry Points)
// ============================================================================

// GetDiscoveryInfo Fitur "Dream/Inspiration"
func (p *AIPlanner) GetDiscoveryInfo(ctx context.Context, city string) (*domain.DiscoveryResponse, error) {
	// ... (unchanged)
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
	cleanData := cleanJSON(rawResponse)

	var resp domain.DiscoveryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("❌ Failed JSON Discovery: %s\n", string(cleanData))
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &resp, nil
}

// EnhanceActivity (M-128): AI-powered enrichment for a single activity
func (p *AIPlanner) EnhanceActivity(ctx context.Context, destination, title string) (*domain.Activity, error) {
	inputData := map[string]string{
		"Destination": destination,
		"Title":       title,
	}

	var rawResponse json.RawMessage
	if err := p.requestAI(ctx, "activity_enrichment", inputData, &rawResponse); err != nil {
		return nil, err
	}

	cleanData := cleanJSON(rawResponse)
	var result struct {
		Description  string   `json:"description"`
		PlaceName    string   `json:"place_name"`
		Latitude     *float64 `json:"latitude"`
		Longitude    *float64 `json:"longitude"`
		Category     string   `json:"category"`
		LocationType string   `json:"location_type"`
	}

	if err := json.Unmarshal(cleanData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI enhancement: %w", err)
	}

	return &domain.Activity{
		Activity:     title,
		Description:  result.Description,
		PlaceName:    result.PlaceName,
		Type:         strings.Title(strings.ToLower(result.Category)),
		Latitude:     result.Latitude,
		Longitude:    result.Longitude,
		LocationType: result.LocationType,
	}, nil
}

// GenerateAddActivitySuggestions (M-128): Suggest activities for a specific bucket
func (p *AIPlanner) GenerateAddActivitySuggestions(ctx context.Context, destination, style, bucket, time string) ([]domain.ActivityAlternative, error) {
	inputData := map[string]string{
		"Destination": destination,
		"Style":       style,
		"Bucket":      bucket,
		"Time":        time,
	}

	var rawResponse json.RawMessage
	if err := p.requestAI(ctx, "add_activity_suggestions", inputData, &rawResponse); err != nil {
		return nil, err
	}

	return p.parseAlternatives(rawResponse)
}

// GenerateOnlyItinerary Fitur "Plan Itinerary"
func (p *AIPlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	var rawResponse json.RawMessage

	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting Itinerary AI Request for %s", trip.Destination)

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

	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		fmt.Printf("⚠️ [AI DEBUG] Itinerary Unmarshal failed: %v\n", err)
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

	if err := p.requestAI(ctx, "planner_logistics_system", trip, &logisticsResp); err != nil {
		log.Printf("❌ Logistics AI Error: %v", err)
		return domain.TripPlan{}, err
	}

	return domain.TripPlan{
		TripID:               trip.ID,
		ArrivalGuide:         &logisticsResp.ArrivalGuide,
		AccommodationOptions: logisticsResp.StrategicAccommodation,
		BudgetBreakdown:      logisticsResp.BudgetBreakdown,
	}, nil
}

// GenerateAlternatives Fitur "Activity Alternatives"
func (p *AIPlanner) GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {
	inputData := map[string]interface{}{
		"Destination": dest,
		"Activity":    activity,
		"Location":    location,
		"Tags":        tags,
	}

	var rawResponse json.RawMessage
	if err := p.requestAI(ctx, "planner_alternatives_system", inputData, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to generate alternatives: %w", err)
	}

	return p.parseAlternatives(rawResponse)
}

// GenerateActivityReplacement (M-128): High-speed alternative generation
func (p *AIPlanner) GenerateActivityReplacement(ctx context.Context, dest, activity string, tags []string) ([]domain.ActivityAlternative, error) {
	inputData := map[string]interface{}{
		"Destination": dest,
		"Activity":    activity,
		"Tags":        tags,
	}

	var rawResponse json.RawMessage
	if err := p.requestAI(ctx, "planner_alternatives_system", inputData, &rawResponse); err != nil {
		return nil, fmt.Errorf("AI replacement failed: %w", err)
	}

	alternatives, err := p.parseAlternatives(rawResponse)
	if err != nil {
		return nil, err
	}

	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}

	return alternatives, nil
}

// --- AI PARSING HELPERS ---

type aiAlt struct {
	Activity     string `json:"activity"`
	Title        string `json:"title"`
	ActivityType string `json:"activity_type"`
	Type         string `json:"type"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	PlaceName    string `json:"place_name"`
}

type aiWrapper struct {
	Alternatives []aiAlt `json:"alternatives"`
	Suggestions  []aiAlt `json:"suggestions"`
	Activities   []aiAlt `json:"activities"`
}

func (p *AIPlanner) parseAlternatives(raw []byte) ([]domain.ActivityAlternative, error) {
	cleanData := cleanJSON(raw)

	var directArray []aiAlt
	if err := json.Unmarshal(cleanData, &directArray); err == nil && len(directArray) > 0 {
		return p.mapToDomainAlternatives(directArray), nil
	}

	var wrapper aiWrapper
	if err := json.Unmarshal(cleanData, &wrapper); err == nil {
		if len(wrapper.Alternatives) > 0 {
			return p.mapToDomainAlternatives(wrapper.Alternatives), nil
		}
		if len(wrapper.Suggestions) > 0 {
			return p.mapToDomainAlternatives(wrapper.Suggestions), nil
		}
		if len(wrapper.Activities) > 0 {
			return p.mapToDomainAlternatives(wrapper.Activities), nil
		}
	}

	return nil, fmt.Errorf("json syntax error: could not parse alternatives from AI response (Raw: %s)", string(cleanData))
}

func (p *AIPlanner) mapToDomainAlternatives(rawAlts []aiAlt) []domain.ActivityAlternative {
	domainAlts := make([]domain.ActivityAlternative, len(rawAlts))
	for i, r := range rawAlts {
		title := r.Activity
		if title == "" {
			title = r.Title
		}
		if title == "" {
			title = r.PlaceName
		}

		aType := r.Type
		if aType == "" {
			aType = r.Category
		}
		if aType == "" {
			aType = r.ActivityType
		}

		domainAlts[i] = domain.ActivityAlternative{
			Activity:    title,
			Type:        strings.Title(strings.ToLower(aType)),
			Description: r.Description,
			PlaceName:   r.PlaceName,
		}
	}
	return domainAlts
}

// GeneratePackingList Membuat daftar bawaan cerdas
func (p *AIPlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingCategory, error) {
	var result struct {
		PackingList []domain.PackingCategory `json:"packing_list"`
	}

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
		return p.generateMockPlan(trip, nil), nil
	}

	// 3. [NEW] Ensure Destination Airport is populated
	destAirport := fullResponse.DestinationAirport
	if destAirport == "" && p.amadeusSvc != nil {
		log.Printf("🔍 Destination Airport missing from AI. Searching Amadeus for '%s'...", trip.Destination)
		locs, err := p.amadeusSvc.SearchLocations(ctx, trip.Destination)
		if err == nil && len(locs) > 0 {
			code := locs[0].IataCode
			destAirport = code
			log.Printf("✅ Found Airport Code: %s", code)
		} else {
			log.Printf("⚠️ Failed to find airport code: %v", err)
		}
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
		DestinationAirport:   destAirport, // Set the populated airport code
	}

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
	// ... (prompt rendering)
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, data)
	if err != nil {
		return fmt.Errorf("render prompt error [%s]: %w", sysKey, err)
	}

	userDataBytes, _ := json.Marshal(data)
	userContent := fmt.Sprintf("Here is the trip context data:\n%s", string(userDataBytes))

	// Dynamically determine ResponseFormat:
	// JSON_OBJECT mode only supports Objects ({...}). It fails for Arrays ([...]).
	// If the target is a slice pointer, we MUST use nil/Text format to allow arrays.
	var respFormat *openai.ChatCompletionResponseFormat
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() == reflect.Ptr && targetVal.Elem().Kind() == reflect.Slice {
		// Target is a slice (e.g. *[]domain.ActivityAlternative).
		// We allow Text/nil format so the model can return a direct array [...].
		respFormat = nil
	} else {
		// Default to JSON_OBJECT for structural safety if target is a struct/object.
		respFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature:    0.7,
		ResponseFormat: respFormat,
	})

	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("openai returned empty choices")
	}

	rawContent := resp.Choices[0].Message.Content
	cleanContent := cleanJSON([]byte(rawContent))

	if err := json.Unmarshal(cleanContent, target); err != nil {
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

// GenerateTripSkeleton (Phase 1): Ultra-fast generation of itinerary structure only
func (p *AIPlanner) GenerateTripSkeleton(ctx context.Context, trip domain.Trip) (domain.ItineraryResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_SKELETON AI Request (Phase 1) for %s...", trip.Destination)

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Trip": trip,
	}

	if err := p.requestAI(ctx, "TRIP_SKELETON", inputData, &rawResponse); err != nil {
		return domain.ItineraryResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_SKELETON Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("skeleton parse error: %w", err)
	}

	// Flag activities as skeleton to trigger frontend lazy-loading
	for i := range resp.Itinerary {
		for j := range resp.Itinerary[i].Activities {
			resp.Itinerary[i].Activities[j].IsSkeleton = true
		}
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

// GenerateTripOverview (Phase 1): Fast-track generation of high-level details
func (p *AIPlanner) GenerateTripOverview(ctx context.Context, trip domain.Trip) (domain.TripOverviewResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_OVERVIEW AI Request (Phase 1) for %s...", trip.Destination)

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Trip":        trip,
		"Destination": trip.Destination,
		"TripDays":    trip.TripDays,
		"Style":       trip.Style,
		"Budget":      trip.Budget,
	}

	if err := p.requestAI(ctx, "TRIP_OVERVIEW", inputData, &rawResponse); err != nil {
		return domain.TripOverviewResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_OVERVIEW Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.TripOverviewResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.TripOverviewResponse{}, fmt.Errorf("overview parse error: %w", err)
	}

	return resp, nil
}

// GenerateTripItinerary (Phase 2): Detailed schedule generation based on overview context
func (p *AIPlanner) GenerateTripItinerary(ctx context.Context, trip domain.Trip, overviewJSON string) (domain.ItineraryResponse, error) {
	startTime := time.Now()
	log.Printf("⏱️ [PERF] Starting TRIP_ITINERARY AI Request (Phase 2) for %s...", trip.Destination)

	var rawResponse json.RawMessage
	inputData := map[string]interface{}{
		"Trip":         trip,
		"OverviewJSON": overviewJSON,
	}

	if err := p.requestAI(ctx, "TRIP_ITINERARY", inputData, &rawResponse); err != nil {
		return domain.ItineraryResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_ITINERARY Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("itinerary parse error: %w", err)
	}

	return resp, nil
}

// GetRegeneratePrompt Generates the text prompt context without calling the AI (For QA/Testing)
func (p *AIPlanner) GetRegeneratePrompt(ctx context.Context, trip domain.Trip, prefs domain.UserPreferences) (string, error) {
	// 1. Prepare Constraint Strings
	var constraints []string

	// Dietary
	if len(prefs.Dietary) > 0 {
		constraints = append(constraints, fmt.Sprintf("CRITICAL CONSTRAINT: The user follows these dietary restrictions: %s. All restaurant suggestions MUST strictly follow this.", strings.Join(prefs.Dietary, ", ")))
	}

	// Pace
	switch prefs.Pace {
	case "FAST":
		constraints = append(constraints, "Pace: High Intensity. Pack as many activities as possible (4-5 per day). Minimize gaps.")
	case "RELAXED":
		constraints = append(constraints, "Pace: Relaxed. Limit to 2-3 major activities per day. Allow ample time for leisure and transit.")
	}

	// Vibe / Interests
	if len(prefs.Interests) > 0 {
		constraints = append(constraints, fmt.Sprintf("Vibe Focus: Prioritize activities related to %s.", strings.Join(prefs.Interests, ", ")))
	}

	// 2. Prepare Context (Trip + DNA + Constraints)
	tripContext := map[string]interface{}{
		"Trip":        trip,
		"Preferences": prefs,
		"Constraints": strings.Join(constraints, "\n"),
	}

	// 3. Render System Prompt (Rules & Schema)
	// We use the same system key as GeneratePlan
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, "planner_itinerary_system", tripContext)
	if err != nil {
		return "", fmt.Errorf("render prompt error: %w", err)
	}

	// 4. Return the Final Prompt (System + User Data)
	userDataBytes, _ := json.Marshal(tripContext)
	fullPrompt := fmt.Sprintf("--- SYSTEM ---\n%s\n\n--- USER DATA ---\n%s", sysPrompt, string(userDataBytes))
	return fullPrompt, nil
}
