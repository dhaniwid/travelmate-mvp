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

// Konstanta
const (
	ModelFast = openai.GPT4oMini
	ModelRich = openai.GPT4o
)

type AIPlanner struct {
	client    *openai.Client
	promptSvc *PromptService
}

type PlannerPromptData struct {
	Days             int
	Destination      string
	Origin           string
	Style            string
	Budget           string
	StartDate        string
	TransportContext string
}

func NewAIPlanner(apiKey string, promptSvc *PromptService) *AIPlanner {
	client := openai.NewClient(apiKey)
	return &AIPlanner{client: client, promptSvc: promptSvc}
}

func (p *AIPlanner) requestOpenAI(ctx context.Context, sysKey string, trip domain.Trip, target interface{}) error {
	sysPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, sysKey, nil)
	if err != nil {
		return fmt.Errorf("system prompt error: %w", err)
	}

	userPrompt, err := p.promptSvc.GetRenderedPrompt(ctx, "planner_user", trip)
	if err != nil {
		return fmt.Errorf("user prompt error: %w", err)
	}

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

	return p.parseJSON(resp.Choices[0].Message.Content, target)
}

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

func (p *AIPlanner) GenerateOnlyItinerary(ctx context.Context, trip domain.Trip) ([]domain.ItineraryDay, error) {
	var result struct {
		Itinerary []domain.ItineraryDay `json:"itinerary"`
	}

	err := p.requestOpenAI(ctx, "planner_itinerary_system", trip, &result)
	return result.Itinerary, err
}

func (p *AIPlanner) GenerateTransportAndStay(ctx context.Context, trip domain.Trip) (domain.TripPlan, error) {
	var aiResp domain.AIPlannerResponse

	err := p.requestOpenAI(ctx, "planner_logistics_system", trip, &aiResp)
	if err != nil {
		return domain.TripPlan{}, err
	}

	return domain.TripPlan{
		TransportOptions:     aiResp.TransportOptions,
		AccommodationOptions: aiResp.AccommodationOptions,
		BudgetBreakdown:      aiResp.BudgetBreakdown,
	}, nil
}

func (p *AIPlanner) cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")
	return strings.TrimSpace(content)
}

// --- HELPER: JSON PARSER ---
func (p *AIPlanner) parseJSON(content string, target interface{}) error {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return json.Unmarshal([]byte(content), target)
}

// --- HELPER: MOCK / FALLBACK PLAN GENERATOR ---
func (s *AIPlanner) generateMockPlan(req domain.Trip, realTickets []domain.TransportOption) domain.TripPlan {
	log.Println("⚠️ OpenAI Error or JSON Invalid. Switching to Mock Plan.")

	// 1. Generate Dummy Itinerary (Sesuai format baru: []Activity)
	itinerary := []domain.ItineraryDay{}
	for i := 1; i <= req.TripDays; i++ {
		day := domain.ItineraryDay{Day: i}

		if i == 1 {
			day.Title = "Arrival & Settlement"
			day.Activities = []domain.Activity{
				{Time: "14:00", Activity: "Check-in Hotel", Type: "Logistics", PlaceName: "Hotel", Description: "Check in process"},
				{Time: "16:00", Activity: "City Center Walk", Type: "Sightseeing", PlaceName: "Alun-Alun", Description: "Light walking"},
				{Time: "19:00", Activity: "Local Dinner", Type: "Culinary", PlaceName: "Street Food Center", Description: "Dinner time"},
			}
		} else {
			day.Title = "Exploration Day"
			day.Activities = []domain.Activity{
				{Time: "09:00", Activity: "Main Attraction Visit", Type: "Sightseeing", PlaceName: "Famous Landmark", Description: "Explore the icon"},
				{Time: "13:00", Activity: "Lunch at Local Resto", Type: "Culinary", PlaceName: "Legendary Resto", Description: "Lunch break"},
				{Time: "16:00", Activity: "Shopping / Relax", Type: "Leisure", PlaceName: "Souvenir Shop", Description: "Free time"},
			}
		}
		itinerary = append(itinerary, day)
	}

	// 2. Dummy Transport (Jika tidak ada tiket real)
	transportOpts := realTickets
	if len(transportOpts) == 0 {
		transportOpts = []domain.TransportOption{
			{Type: "Flight", Name: "Mock Air", Price: 1500000, EstimatedTime: "1h 30m", Pros: "Fastest"},
			{Type: "Train", Name: "Mock Express", Price: 400000, EstimatedTime: "5h 00m", Pros: "Scenic"},
		}
	}

	// 3. Dummy Accommodation
	accomOpts := []domain.AccommodationOption{
		{Name: "Mock Luxury Hotel", Type: "Hotel", Rating: "5.0", PricePerNight: 2000000, LocationArea: "City Center", Description: "Best in town"},
		{Name: "Mock Budget Hostel", Type: "Hostel", Rating: "4.2", PricePerNight: 250000, LocationArea: "Downtown", Description: "Backpacker choice"},
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
