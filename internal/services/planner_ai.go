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

	err := p.requestOpenAI(ctx, "planner_system", trip, &aiResp)
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
	var result struct {
		Itinerary []domain.ItineraryDay `json:"itinerary"`
	}

	err := p.requestOpenAI(ctx, "planner_itinerary_system", trip, &result)
	return result.Itinerary, err
}

// GenerateTransportAndStay: Hanya membuat Logistik (Transport & Hotel)
// Termasuk Auto-Correction untuk masalah mata uang (JPY/USD -> IDR)
func (p *AIPlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	var aiResp domain.AIPlannerResponse

	// 1. Request ke AI
	err := p.requestOpenAI(ctx, "planner_logistics_system", trip, &aiResp)
	if err != nil {
		return domain.TripPlan{}, err
	}

	fmt.Printf("🛎️ AI Logistics Response: %+v\n", aiResp)

	// 2. Sanity Check & Auto-Correct Currency
	// (Dipisahkan agar kode bersih)
	p.sanitizeLogisticsPrices(&aiResp, trip.TripDays)

	return domain.TripPlan{
		TransportOptions:     aiResp.TransportOptions,
		AccommodationOptions: aiResp.AccommodationOptions,
		BudgetBreakdown:      aiResp.BudgetBreakdown,
	}, nil
}

// ============================================================================
// PRIVATE HELPER METHODS (Logic Internals)
// ============================================================================

// requestOpenAI: Centralized logic untuk memanggil OpenAI API
func (p *AIPlanner) requestOpenAI(ctx context.Context, sysKey string, trip domain.Trip, target interface{}) error {
	// 1. Prepare Prompts
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, nil)
	if err != nil {
		return fmt.Errorf("system prompt error: %w", err)
	}

	userPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, "planner_user", trip)
	if err != nil {
		return fmt.Errorf("user prompt error: %w", err)
	}

	// 2. Call OpenAI
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: ModelFast,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.7, // Agak kreatif tapi tetap terkontrol
	})

	if err != nil {
		return fmt.Errorf("openai api error: %w", err)
	}

	// 3. Parse Response
	return p.parseJSON(resp.Choices[0].Message.Content, target)
}

// sanitizeLogisticsPrices: Memperbaiki harga hotel yang tidak masuk akal (deteksi mata uang asing)
func (p *AIPlanner) sanitizeLogisticsPrices(aiResp *domain.AIPlannerResponse, tripDays int) {
	const ThresholdIDR = 100000 // Batas bawah harga wajar (100rb)
	const YenToIDR = 105        // Kurs kasar 1 JPY = 105 IDR

	for i, acc := range aiResp.AccommodationOptions {
		price := int64(acc.PricePerNight) // Cast FlexibleInt64 ke int64

		// Deteksi harga mencurigakan (contoh: 8.000 atau 15.000)
		if price > 0 && price < ThresholdIDR {
			// Jika range harganya ribuan (1.000 - 50.000), kemungkinan besar itu JPY/THB/PHP
			if price >= 1000 && price <= 50000 {
				newPrice := price * YenToIDR
				log.Printf("⚠️ Detected suspicious hotel price (%d). Auto-correcting JPY to IDR: %d", price, newPrice)

				// Update Harga Hotel
				aiResp.AccommodationOptions[i].PricePerNight = int64(domain.FlexibleInt64(newPrice))

				// Update Total Budget Breakdown agar sinkron
				currentAccomBudget := int64(aiResp.BudgetBreakdown.Accommodation)
				diff := (newPrice * int64(tripDays)) - (price * int64(tripDays))
				aiResp.BudgetBreakdown.Accommodation = domain.FlexibleInt64(currentAccomBudget + diff)
			}
		}
	}
}

// parseJSON: Membersihkan Markdown code block sebelum unmarshal
func (p *AIPlanner) parseJSON(content string, target interface{}) error {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return json.Unmarshal([]byte(content), target)
}

// ============================================================================
// MOCK / FALLBACK LOGIC
// ============================================================================

func (s *AIPlanner) generateMockPlan(req domain.Trip, realTickets []domain.TransportOption) domain.TripPlan {
	log.Println("⚠️ OpenAI Error or JSON Invalid. Switching to Mock Plan.")

	// 1. Dummy Itinerary
	itinerary := []domain.ItineraryDay{}
	for i := 1; i <= req.TripDays; i++ {
		day := domain.ItineraryDay{Day: i}
		if i == 1 {
			day.Title = "Arrival & Settlement"
			day.Activities = []domain.Activity{
				{Time: "14:00", Activity: "Check-in Hotel", Type: "Logistics", PlaceName: "Hotel", Description: "Check in process"},
				{Time: "19:00", Activity: "Local Dinner", Type: "Culinary", PlaceName: "Street Food Center", Description: "Dinner time"},
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

	// 2. Dummy Transport
	transportOpts := realTickets
	if len(transportOpts) == 0 {
		transportOpts = []domain.TransportOption{
			{Type: "Flight (Mock)", Name: "Mock Air", Price: 1500000, EstimatedTime: "1h 30m", Pros: "Fastest"},
		}
	}

	// 3. Dummy Accommodation
	accomOpts := []domain.AccommodationOption{
		{Name: "Mock Luxury Hotel", Type: "Hotel", Rating: "5.0", PricePerNight: 2000000, LocationArea: "City Center", Description: "Best in town"},
	}

	// 4. Dummy Budget
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
		DecisionNotes:        []string{"⚠️ This is a generated mock plan because AI service is unavailable."},
	}
}
