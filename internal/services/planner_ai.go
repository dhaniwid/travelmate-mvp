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
	client        *openai.Client
	promptSvc     *PromptService
	prefRepo      *repositories.PreferencesRepository
	amadeusSvc    *AmadeusService
	knowledgeRepo KnowledgeSearcher // RAG: local knowledge retrieval
}

// KnowledgeSearcher is a thin interface over KnowledgeRepository for testability.
type KnowledgeSearcher interface {
	SearchSimilarKnowledge(ctx context.Context, city string, queryEmbedding []float32, limit int) ([]domain.LocalKnowledge, error)
}

func NewAIPlanner(apiKey string, promptSvc *PromptService, prefRepo *repositories.PreferencesRepository, amadeusSvc *AmadeusService, knowledgeRepo KnowledgeSearcher) *AIPlanner {
	client := openai.NewClient(apiKey)
	return &AIPlanner{
		client:        client,
		promptSvc:     promptSvc,
		prefRepo:      prefRepo,
		amadeusSvc:    amadeusSvc,
		knowledgeRepo: knowledgeRepo,
	}
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

	// ── RAG INJECTION ── Inject local insider knowledge into editorial generation
	ragContext := p.fetchLocalKnowledge(ctx, trip.Destination, trip.Style, trip.TripDays)

	if err := p.requestAIWithRAG(ctx, "planner_editorial_system", trip, ragContext, &rawResponse, 0); err != nil {
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
	if err := validateEditorialResponse(resp, "GenerateEditorial"); err != nil {
		return domain.EditorialResponse{}, fmt.Errorf("editorial validation failed: %w", err)
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
		alts := p.mapToDomainAlternatives(directArray)
		if err := validateActivityAlternatives(alts, "parseAlternatives"); err != nil {
			return nil, err
		}
		return alts, nil
	}

	var wrapper aiWrapper
	if err := json.Unmarshal(cleanData, &wrapper); err == nil {
		var alts []domain.ActivityAlternative
		if len(wrapper.Alternatives) > 0 {
			alts = p.mapToDomainAlternatives(wrapper.Alternatives)
		} else if len(wrapper.Suggestions) > 0 {
			alts = p.mapToDomainAlternatives(wrapper.Suggestions)
		} else if len(wrapper.Activities) > 0 {
			alts = p.mapToDomainAlternatives(wrapper.Activities)
		}
		if len(alts) > 0 {
			if err := validateActivityAlternatives(alts, "parseAlternatives"); err != nil {
				return nil, err
			}
			return alts, nil
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
	if err := validateAIPlannerResponse(fullResponse, "GeneratePlan"); err != nil {
		fmt.Printf("❌ [AI VALIDATION] Full Plan Invalid: %v\n", err)
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

// requestAIWithRAG wraps requestAI with an extra RAG knowledge context message.
// ragContext is a pre-formatted string block such as:
//
//	🧠 LOCAL KNOWLEDGE — Bandung:
//	1. [cafe] Warung Kopi X — Hidden gem near Dago ...
//
// If ragContext is empty, this behaves identically to requestAI.
// requestAIWithRAG wraps requestAI with an extra RAG knowledge context message.
// maxTokens: 0 = no limit (model default). Pass a value e.g. 2500 to cap output tokens.
func (p *AIPlanner) requestAIWithRAG(ctx context.Context, sysKey string, data interface{}, ragContext string, target interface{}, maxTokens int) error {
	t0 := time.Now()

	promptStart := time.Now()
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, data)
	if err != nil {
		return fmt.Errorf("render prompt error [%s]: %w", sysKey, err)
	}
	log.Printf("⏱️ [GEN:%s] GetRenderedPrompt: %dms", sysKey, time.Since(promptStart).Milliseconds())

	// If RAG data exists, prepend it to the system prompt so the model always
	// has local intel regardless of context window ordering.
	if ragContext != "" {
		sysPrompt = ragContext + "\n\n" + sysPrompt
		log.Printf("🧠 [RAG] Injected local knowledge block (%d bytes) into system prompt for [%s]", len(ragContext), sysKey)
	}

	userDataBytes, _ := json.Marshal(data)
	userContent := fmt.Sprintf("Here is the trip context data:\n%s", string(userDataBytes))

	var respFormat *openai.ChatCompletionResponseFormat
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() == reflect.Ptr && targetVal.Elem().Kind() == reflect.Slice {
		respFormat = nil
	} else {
		respFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	}

	req := openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature:    0.7,
		ResponseFormat: respFormat,
	}
	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	}

	aiStart := time.Now()
	log.Printf("⏱️ [GEN:%s] OpenAI call sent (maxTokens=%d)", sysKey, maxTokens)
	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}
	log.Printf("⏱️ [GEN:%s] OpenAI response received: %dms | prompt_tokens=%d completion_tokens=%d total_tokens=%d",
		sysKey,
		time.Since(aiStart).Milliseconds(),
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens,
	)

	if len(resp.Choices) == 0 {
		return fmt.Errorf("openai returned empty choices")
	}

	parseStart := time.Now()
	rawContent := resp.Choices[0].Message.Content
	cleanContent := cleanJSON([]byte(rawContent))
	if err := json.Unmarshal(cleanContent, target); err != nil {
		fmt.Printf("❌ JSON Syntax Error for [%s+RAG]. \nContent: %s\n", sysKey, cleanContent)
		return fmt.Errorf("json syntax error: %w", err)
	}
	log.Printf("⏱️ [GEN:%s] Response parsed: %dms | total requestAIWithRAG: %dms", sysKey, time.Since(parseStart).Milliseconds(), time.Since(t0).Milliseconds())

	return nil
}

// fetchLocalKnowledge generates a query embedding for the trip destination + style,
// retrieves the top-5 most similar local knowledge items from pgvector, and returns
// them as a formatted string block ready for system prompt injection.
//
// Failure is non-fatal: if the search fails or returns 0 results, an empty string
// is returned and trip generation continues normally.
func (p *AIPlanner) fetchLocalKnowledge(ctx context.Context, city, style string, tripDays int) string {
	if p.knowledgeRepo == nil {
		return ""
	}

	// Build a semantic query that mirrors the ingestion input format
	queryText := fmt.Sprintf("City: %s, travel style: %s, %d days trip", city, style, tripDays)

	// Step 1: Generate query embedding with text-embedding-3-small
	embResp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{queryText},
		Model: openai.SmallEmbedding3,
	})
	if err != nil {
		log.Printf("⚠️ [RAG] Embedding generation failed for %q: %v. Continuing without local knowledge.", city, err)
		return ""
	}
	if len(embResp.Data) == 0 {
		log.Printf("⚠️ [RAG] OpenAI returned empty embedding data. Continuing without local knowledge.")
		return ""
	}
	queryEmbedding := embResp.Data[0].Embedding

	// Step 2: Vector similarity search — top 5 most relevant local facts
	knowledgeItems, err := p.knowledgeRepo.SearchSimilarKnowledge(ctx, city, queryEmbedding, 5)
	if err != nil {
		log.Printf("⚠️ [RAG] Vector search failed for city=%q: %v. Continuing without local knowledge.", city, err)
		return ""
	}
	if len(knowledgeItems) == 0 {
		log.Printf("ℹ️ [RAG] No local knowledge found for city=%q. Continuing normally.", city)
		return ""
	}

	log.Printf("✅ [RAG] Retrieved %d local knowledge items for %q", len(knowledgeItems), city)
	return formatKnowledgeBlock(city, knowledgeItems)
}

// formatKnowledgeBlock renders retrieved knowledge items into a clean prompt-injectable block.
func formatKnowledgeBlock(city string, items []domain.LocalKnowledge) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🧠 LOCAL INSIDER KNOWLEDGE — %s:\n", city))
	sb.WriteString("Use the following hyper-local facts to recommend authentic, off-the-beaten-path experiences. Prioritize these over generic tourist spots where relevant.\n")
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s — %s\n", i+1, item.Category, item.Name, item.Description))
	}
	return sb.String()
}

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
	log.Printf("⏱️ [GEN] Start — %s (%d days)", trip.Destination, trip.TripDays)

	inputData := map[string]interface{}{
		"Trip": trip,
	}

	// ── RAG INJECTION ── Retrieve local knowledge for this destination
	ragStart := time.Now()
	ragContext := p.fetchLocalKnowledge(ctx, trip.Destination, trip.Style, trip.TripDays)
	log.Printf("⏱️ [GEN] FetchLocalKnowledge (RAG embedding + pgvector): %dms", time.Since(ragStart).Milliseconds())

	const maxRetries = 3
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("🔄 [RETRY] TRIP_SKELETON attempt %d/%d for %s", attempt, maxRetries, trip.Destination)
		}

		var rawResponse json.RawMessage
		// requestAIWithRAG logs: GetRenderedPrompt, OpenAI sent, OpenAI received (with token counts), Response parsed
		if err := p.requestAIWithRAG(ctx, "TRIP_SKELETON", inputData, ragContext, &rawResponse, 2500); err != nil {
			lastErr = err
			continue
		}

		cleanData := cleanJSON(rawResponse)
		if len(cleanData) < 2000 {
			log.Printf("🔍 [SKELETON RAW] attempt %d: %s", attempt, string(cleanData))
		} else {
			log.Printf("🔍 [SKELETON RAW] attempt %d (truncated): %s...", attempt, string(cleanData[:2000]))
		}

		var resp domain.ItineraryResponse
		if err := json.Unmarshal(cleanData, &resp); err != nil {
			lastErr = fmt.Errorf("skeleton parse error: %w", err)
			continue
		}

		// Fallback: if activity title is empty but place_name exists, use place_name
		for i := range resp.Itinerary {
			for j := range resp.Itinerary[i].Activities {
				act := &resp.Itinerary[i].Activities[j]
				if strings.TrimSpace(act.Activity) == "" && strings.TrimSpace(act.PlaceName) != "" {
					act.Activity = act.PlaceName
				}
				// Last resort: generate placeholder so validation never hard-fails
				if strings.TrimSpace(act.Activity) == "" {
					act.Activity = fmt.Sprintf("Aktivitas %s", act.Time)
					log.Printf("⚠️ [SKELETON] Day %d act %d: used placeholder title", i+1, j+1)
				}
			}
		}

		valStart := time.Now()
		if err := validateItineraryResponse(resp, "GenerateTripSkeleton"); err != nil {
			lastErr = fmt.Errorf("skeleton validation failed: %w", err)
			log.Printf("❌ [SKELETON] Validation error on attempt %d: %v", attempt, err)
			continue
		}
		log.Printf("⏱️ [GEN] Validation: %dms", time.Since(valStart).Milliseconds())

		// Flag activities as skeleton to trigger frontend lazy-loading
		for i := range resp.Itinerary {
			for j := range resp.Itinerary[i].Activities {
				resp.Itinerary[i].Activities[j].IsSkeleton = true
			}
		}

		log.Printf("⏱️ [GEN] GenerateTripSkeleton total (excl. DB): %dms", time.Since(startTime).Milliseconds())
		return resp, nil
	}

	return domain.ItineraryResponse{}, fmt.Errorf("skeleton generation failed after %d attempts: %w", maxRetries, lastErr)
}

// FetchRAGContext is the public wrapper around fetchLocalKnowledge for use by other service methods.
func (p *AIPlanner) FetchRAGContext(ctx context.Context, city, style string, tripDays int) string {
	return p.fetchLocalKnowledge(ctx, city, style, tripDays)
}

// GenerateSkeletonStreaming is the streaming variant of GenerateTripSkeleton.
// It accepts a pre-fetched ragContext so RAG and streaming can be parallelised by the caller.
// Returns the parsed ItineraryResponse once all chunks have been received.
func (p *AIPlanner) GenerateSkeletonStreaming(ctx context.Context, trip domain.Trip, ragContext string) (domain.ItineraryResponse, error) {
	startTime := time.Now()

	inputData := map[string]interface{}{"Trip": trip}

	promptStart := time.Now()
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, "TRIP_SKELETON", inputData)
	if err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("render prompt error: %w", err)
	}
	if ragContext != "" {
		sysPrompt = ragContext + "\n\n" + sysPrompt
	}
	log.Printf("⏱️ [GEN:STREAM] GetRenderedPrompt: %dms", time.Since(promptStart).Milliseconds())

	userDataBytes, _ := json.Marshal(inputData)
	userContent := fmt.Sprintf("Here is the trip context data:\n%s", string(userDataBytes))

	req := openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature:    0.7,
		MaxTokens:      2500,
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	}

	aiStart := time.Now()
	log.Printf("⏱️ [GEN:STREAM] OpenAI stream started")
	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("stream create error: %w", err)
	}
	defer stream.Close()

	var fullContent strings.Builder
	firstChunk := true
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return domain.ItineraryResponse{}, fmt.Errorf("stream recv error: %w", err)
		}
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" && firstChunk {
				log.Printf("⏱️ [GEN:STREAM] First chunk received: %dms", time.Since(aiStart).Milliseconds())
				firstChunk = false
			}
			fullContent.WriteString(delta)
		}
	}
	log.Printf("⏱️ [GEN:STREAM] Stream complete: %dms | content_len=%d", time.Since(aiStart).Milliseconds(), fullContent.Len())

	cleanData := cleanJSON([]byte(fullContent.String()))
	if len(cleanData) < 2000 {
		log.Printf("🔍 [STREAM RAW]: %s", string(cleanData))
	} else {
		log.Printf("🔍 [STREAM RAW] (truncated): %s...", string(cleanData[:2000]))
	}

	var resp domain.ItineraryResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("parse error: %w", err)
	}

	// Fallback for empty activity titles
	for i := range resp.Itinerary {
		for j := range resp.Itinerary[i].Activities {
			act := &resp.Itinerary[i].Activities[j]
			if strings.TrimSpace(act.Activity) == "" && strings.TrimSpace(act.PlaceName) != "" {
				act.Activity = act.PlaceName
			}
			if strings.TrimSpace(act.Activity) == "" {
				act.Activity = fmt.Sprintf("Aktivitas %s", act.Time)
			}
		}
	}

	if err := validateItineraryResponse(resp, "GenerateSkeletonStreaming"); err != nil {
		return domain.ItineraryResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	for i := range resp.Itinerary {
		for j := range resp.Itinerary[i].Activities {
			resp.Itinerary[i].Activities[j].IsSkeleton = true
		}
	}

	log.Printf("⏱️ [GEN:STREAM] GenerateSkeletonStreaming total: %dms", time.Since(startTime).Milliseconds())
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

	// ── RAG INJECTION ── Local knowledge improves accommodation area recommendations
	// and transport operator hints with destination-specific context.
	ragContext := p.fetchLocalKnowledge(ctx, trip.Destination, trip.Style, trip.TripDays)

	if err := p.requestAIWithRAG(ctx, "TRIP_LOGISTICS", inputData, ragContext, &rawResponse, 0); err != nil {
		return domain.TripLogisticsResponse{}, err
	}

	log.Printf("⏱️ [PERF] TRIP_LOGISTICS Request completed in: %v", time.Since(startTime))

	cleanData := cleanJSON(rawResponse)
	var resp domain.TripLogisticsResponse
	if err := json.Unmarshal(cleanData, &resp); err != nil {
		return domain.TripLogisticsResponse{}, fmt.Errorf("logistics parse error: %w", err)
	}
	if err := validateLogisticsResponse(resp, "GenerateTripLogistics"); err != nil {
		return domain.TripLogisticsResponse{}, fmt.Errorf("logistics validation failed: %w", err)
	}

	return resp, nil
}

// GenerateTransportOnDemand generates transport_options given an origin city (MT-79).
func (p *AIPlanner) GenerateTransportOnDemand(ctx context.Context, originCity, destination string, tripDays int) ([]domain.TransportOption, error) {
	inputData := map[string]interface{}{
		"OriginCity":  originCity,
		"Destination": destination,
		"TripDays":    tripDays,
	}

	var result struct {
		TransportOptions []domain.TransportOption `json:"transport_options"`
	}
	if err := p.requestAI(ctx, "TRIP_TRANSPORT", inputData, &result); err != nil {
		return nil, fmt.Errorf("TRIP_TRANSPORT ai error: %w", err)
	}
	return result.TransportOptions, nil
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
