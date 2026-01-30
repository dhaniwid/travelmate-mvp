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

// GeneratePlan: Membuat Full Plan (Itinerary + Logistics + Budget)
// Jika gagal, akan fallback ke Mock Plan.
func (p *AIPlanner) GeneratePlan(ctx context.Context, trip domain.Trip, tickets []domain.TransportOption) (domain.TripPlan, error) {
	var aiResp domain.AIPlannerResponse

	err := p.requestOpenAIWithRoleUser(ctx, "planner_system", trip, &aiResp)
	if err != nil {
		log.Printf("❌ Full Plan Error: %v. Switching to Mock.", err)
		return p.generateMockPlan(trip, tickets), nil
	}

	return domain.TripPlan{
		TripID:               trip.ID,
		Itinerary:            aiResp.Itinerary,
		BudgetBreakdown:      aiResp.BudgetBreakdown,
		TransportOptions:     aiResp.TransportOptions,
		AccommodationOptions: aiResp.AccommodationOptions,
		DecisionNotes:        aiResp.DecisionNotes,
	}, nil
}

// GenerateOnlyItinerary: Hanya membuat jadwal harian (Optimized for Parallel Stream)
func (p *AIPlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error) {
	// 1. Ambil Raw Response sebagai string/bytes dulu
	var raw json.RawMessage

	// Pastikan requestOpenAI mendukung pointer ke json.RawMessage
	if err := p.requestOpenAIWithRoleUser(ctx, "planner_itinerary_system", trip, &raw); err != nil {
		return nil, err
	}

	// 2. Bersihkan Markdown (```json ... ```) jika ada
	cleanData := cleanJSON(raw)

	// 🔍 DEBUG: Lihat apa yang sebenarnya mau di-parse
	// fmt.Printf("🧹 Clean JSON for Itinerary: %s\n", string(cleanData))

	// 3. STRATEGI A: Coba Parse sebagai Object Wrapper { "itinerary": [...] }
	var wrapper struct {
		Itinerary []domain.ItineraryDay `json:"itinerary"`
	}
	if err := json.Unmarshal(cleanData, &wrapper); err == nil {
		if len(wrapper.Itinerary) > 0 {
			return wrapper.Itinerary, nil
		}
	}

	// 4. STRATEGI B: Coba Parse sebagai Array Langsung [...]
	var directArray []domain.ItineraryDay
	if err := json.Unmarshal(cleanData, &directArray); err == nil {
		if len(directArray) > 0 {
			return directArray, nil
		}
	}

	// Jika sampai sini, berarti gagal parse atau datanya kosong
	fmt.Println("❌ Failed to parse Itinerary (Zero length or Invalid JSON structure)")
	return nil, fmt.Errorf("failed to parse itinerary")
}

// GenerateTransportAndStay: Hanya membuat Logistik (Transport & Hotel)
func (p *AIPlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	var aiResp domain.AIPlannerResponse

	// 1. Request ke AI
	err := p.requestOpenAIRoleSystem(ctx, "planner_logistics_system", trip, &aiResp)
	if err != nil {
		return domain.TripPlan{}, err
	}

	fmt.Printf("🛎️ AI Logistics Response: %+v\n", aiResp)

	// 2. Sanity Check & Auto-Correct Currency
	//p.sanitizeLogisticsPrices(&aiResp, trip.TripDays)

	return domain.TripPlan{
		TransportOptions:     aiResp.TransportOptions,
		AccommodationOptions: aiResp.AccommodationOptions,
	}, nil
}

// ----------------------------------------------------------------------------
// NEW FEATURE: Generate Alternatives (Premium)
// ----------------------------------------------------------------------------

// GenerateAlternatives meminta AI untuk memberikan opsi aktivitas lain berdasarkan preference tags
func (p *AIPlanner) GenerateAlternatives(ctx context.Context, dest, activity, location string, tags []string) ([]domain.ActivityAlternative, error) {

	// 1. Ambil Prompt dari Database via PromptService
	// (Pastikan method GetAlternativesPrompt sudah ada di PromptService sesuai diskusi sebelumnya)
	promptText, err := p.promptSvc.GetAlternativesPrompt(ctx, dest, activity, location, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to render alternative prompt: %w", err)
	}

	// 2. Request ke OpenAI (Non-Streaming karena butuh JSON utuh)
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: ModelFast, // Gunakan model hemat
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful travel assistant that outputs strict JSON arrays only.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: promptText,
			},
		},
		Temperature: 0.7, // Sedikit kreatif untuk variasi
	})

	if err != nil {
		return nil, fmt.Errorf("openai alternatives error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	// 3. Parse Response menggunakan helper yang sudah ada
	// Kita reuse p.parseJSON yang sudah menghandle pembersihan markdown
	var alternatives []domain.ActivityAlternative
	if err := p.parseJSON(resp.Choices[0].Message.Content, &alternatives); err != nil {
		return nil, fmt.Errorf("failed to parse alternatives json: %w", err)
	}

	return alternatives, nil
}

// ============================================================================
// PRIVATE HELPER METHODS (Logic Internals)
// ============================================================================

// requestOpenAI: Centralized logic untuk memanggil OpenAI API
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

func (p *AIPlanner) requestOpenAIRoleSystem(ctx context.Context, sysKey string, trip domain.Trip, target interface{}) error {
	// 1. Prepare Prompts
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, nil)
	if err != nil {
		return fmt.Errorf("system prompt error: %w", err)
	}

	fmt.Printf("🤖 Asking OpenAI Role System Only [%s]...\n", sysKey)

	// 2. Call OpenAI
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: ModelFast,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		},
		Temperature: 0.7,
	})

	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}

	rawContent := resp.Choices[0].Message.Content

	// 3. CLEAN & PARSE
	cleanContent := cleanJSON([]byte(rawContent))

	fmt.Printf("🤖 Raw Content for [%s]: %s\n", sysKey, rawContent)

	if err := json.Unmarshal([]byte(cleanContent), target); err != nil {
		// Log raw content jika error, agar mudah debug
		fmt.Printf("❌ JSON Parse Error for [%s]. Content: %s\n", sysKey, cleanContent)
		return fmt.Errorf("json parse error: %w", err)
	}

	return nil
}

func (p *AIPlanner) requestOpenAIWithRoleUser(ctx context.Context, sysKey string, trip domain.Trip, target interface{}) error {
	// 1. Prepare Prompts
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, nil)
	if err != nil {
		return fmt.Errorf("system prompt error: %w", err)
	}

	userPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, "planner_user", trip)
	if err != nil {
		return fmt.Errorf("user prompt error: %w", err)
	}

	fmt.Printf("🤖 Asking OpenAI Role System & User [%s]...\n", sysKey)

	// 2. Call OpenAI
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: ModelFast,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.7,
	})

	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}

	rawContent := resp.Choices[0].Message.Content

	// 3. CLEAN & PARSE
	cleanContent := cleanJSON([]byte(rawContent))

	fmt.Printf("🤖 Raw Content for [%s]: %s\n", sysKey, rawContent)

	if err := json.Unmarshal([]byte(cleanContent), target); err != nil {
		// Log raw content jika error, agar mudah debug
		fmt.Printf("❌ JSON Parse Error for [%s]. Content: %s\n", sysKey, cleanContent)
		return fmt.Errorf("json parse error: %w", err)
	}

	return nil
}

// sanitizeLogisticsPrices: Memperbaiki harga hotel yang tidak masuk akal
//func (p *AIPlanner) sanitizeLogisticsPrices(aiResp *domain.AIPlannerResponse, tripDays int) {
//	const ThresholdIDR = 100000
//	const YenToIDR = 105
//
//	for i, acc := range aiResp.AccommodationOptions {
//		price := int64(acc.PricePerNight)
//
//		if price > 0 && price < ThresholdIDR {
//			if price >= 1000 && price <= 50000 {
//				newPrice := price * YenToIDR
//				log.Printf("⚠️ Detected suspicious hotel price (%d). Auto-correcting JPY to IDR: %d", price, newPrice)
//
//				aiResp.AccommodationOptions[i].PricePerNight = int64(domain.FlexibleInt64(newPrice))
//
//				currentAccomBudget := int64(aiResp.BudgetBreakdown.Accommodation)
//				diff := (newPrice * int64(tripDays)) - (price * int64(tripDays))
//				aiResp.BudgetBreakdown.Accommodation = domain.FlexibleInt64(currentAccomBudget + diff)
//			}
//		}
//	}
//}

// parseJSON: Membersihkan Markdown code block sebelum unmarshal
func (p *AIPlanner) parseJSON(content string, target interface{}) error {
	content = strings.TrimSpace(content)
	// Membersihkan ```json dan ```
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}
	content = strings.TrimSpace(content)

	return json.Unmarshal([]byte(content), target)
}

// ============================================================================
// MOCK / FALLBACK LOGIC
// ============================================================================

func (s *AIPlanner) generateMockPlan(req domain.Trip, realTickets []domain.TransportOption) domain.TripPlan {
	log.Println("⚠️ OpenAI Error or JSON Invalid. Switching to Mock Plan.")

	// 1. Dummy Itinerary (Tetap sama, logic sederhana)
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

	// 2. Dummy Transport (SESUAIKAN DENGAN STRUCT BARU)
	// Kita buat 2 opsi: Cepat & Hemat
	transportOpts := []domain.TransportOption{
		{
			StrategyTag:   "CEPAT",
			Name:          "Direct Flight (Mock)",
			PriceTier:     "HIGH",
			EstimatedTime: "1h 30m",
			Pros:          "Fastest way to reach destination (Mock Data).",
			HubDetails: domain.HubDetails{
				DepartureNode: "Origin Airport (CGK)",
				ArrivalNode:   "Destination Airport",
			},
			Breakdown: domain.TransportBreakdown{
				FirstMile: "Taxi to Airport (45m)",
				MainLeg:   "Direct Flight (1h)",
				LastMile:  "Taxi to Hotel (30m)",
			},
		},
		{
			StrategyTag:   "HEMAT",
			Name:          "Intercity Train/Bus (Mock)",
			PriceTier:     "LOW",
			EstimatedTime: "4h 00m",
			Pros:          "Budget friendly option.",
			HubDetails: domain.HubDetails{
				DepartureNode: "Central Station",
				ArrivalNode:   "City Terminal",
			},
			Breakdown: domain.TransportBreakdown{
				FirstMile: "Ojek to Station (30m)",
				MainLeg:   "Economy Train (3h)",
				LastMile:  "Angkot to Area (30m)",
			},
		},
	}

	// 3. Dummy Accommodation (FOKUS AREA & NOTE)
	accomOpts := []domain.AccommodationOption{
		{
			Type:         "Hotel",
			LocationArea: "City Center Zone",
			LocationNote: "Strategic access to all landmarks.",
			Description:  "Mock description: A vibrant area perfect for tourists.",
		},
		{
			Type:         "Villa",
			LocationArea: "Quiet Highlands",
			LocationNote: "Best for relaxation.",
			Description:  "Mock description: Peaceful area away from traffic.",
		},
	}

	// 4. Dummy Context
	logContext := domain.LogisticsContext{
		DistanceKM:   123,
		WarningAlert: "This is a generated MOCK PLAN because the AI service is currently unavailable.",
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
		LogisticsContext:     logContext, // Jangan lupa ini
		DecisionNotes:        []string{"⚠️ This is a generated mock plan because AI service is unavailable."},
	}
}

// GeneratePackingList GeneratePackingList: Membuat daftar bawaan cerdas berdasarkan destinasi, durasi, dan style trip
func (p *AIPlanner) GeneratePackingList(ctx context.Context, trip domain.Trip) ([]domain.PackingItem, error) {
	// 1. Siapkan struct wrapper untuk menangkap JSON output
	var result struct {
		PackingList []domain.PackingItem `json:"packing_list"`
	}

	// 2. Request ke OpenAI
	err := p.requestOpenAIRoleSystem(ctx, "planner_packing_system", trip, &result)
	if err != nil {
		log.Printf("❌ Packing List Generation Error: %v", err)
		return nil, err
	}

	return result.PackingList, nil
}
