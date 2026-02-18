package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"travelmate/internal/repositories"

	openai "github.com/sashabaranov/go-openai"
)

// ChatService handles context-aware AI chat for a specific trip (RAG pattern).
type ChatService struct {
	tripRepo *repositories.TripRepository
	client   *openai.Client
}

func NewChatService(tripRepo *repositories.TripRepository, openAIKey string) *ChatService {
	return &ChatService{
		tripRepo: tripRepo,
		client:   openai.NewClient(openAIKey),
	}
}

// ChatWithTrip loads the trip context and returns an AI reply.
func (s *ChatService) ChatWithTrip(ctx context.Context, tripID, userID, message string) (string, error) {
	// 1. Load Trip Data (Itinerary + Logistics) from DB
	tripAndPlan, err := s.tripRepo.GetTripWithPlan(ctx, tripID)
	if err != nil {
		return "", fmt.Errorf("failed to load trip context: %w", err)
	}
	if tripAndPlan == nil {
		return "", fmt.Errorf("trip not found: %s", tripID)
	}

	// 2. Serialize the plan to JSON for context injection
	planJSON, err := json.Marshal(tripAndPlan.Plan)
	if err != nil {
		return "", fmt.Errorf("failed to serialize trip plan: %w", err)
	}

	destination := tripAndPlan.Trip.Destination

	// 3. Construct System Prompt (RAG) — strict mobile-friendly formatting rules
	systemPrompt := fmt.Sprintf(
		"You are Miru, a friendly local travel expert. "+
			"The user is asking about their trip to %s. "+
			"Here is their current trip plan in JSON format:\n\n%s\n\n"+
			"STRICT RULES — follow these exactly:\n"+
			"• LENGTH: Keep answers under 60 words.\n"+
			"• FORMAT: Use bullet points (•) and relevant emojis to break up text. Never write walls of text.\n"+
			"• TONE: Be a local friend, not a robot. NO apologies. NO 'I am an AI' disclaimers. Be direct and energetic.\n"+
			"• CONTENT: Direct answers only. If suggesting a place, give exactly 1 punchy reason why.\n"+
			"Do NOT output JSON. Respond in natural language with bullets and emojis.",
		destination,
		string(planJSON),
	)

	log.Printf("💬 [Chat] Trip=%s Destination=%s Message=%q", tripID, destination, message)

	// 4. Call OpenAI
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: message},
		},
		Temperature: 0.7,
		MaxTokens:   512,
	})
	if err != nil {
		return "", fmt.Errorf("openai chat error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned empty response")
	}

	reply := resp.Choices[0].Message.Content
	log.Printf("✅ [Chat] Reply generated (%d chars)", len(reply))

	return reply, nil
}
