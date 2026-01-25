package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

type LocationService struct {
	Repo      *repositories.LocationRepository
	AIClient  *openai.Client
	PromptSvc *PromptService
	ImageSvc  *ImageService
}

func NewLocationService(repo *repositories.LocationRepository, promptSvc *PromptService, apiKey string, imgSvc *ImageService) *LocationService {
	return &LocationService{
		Repo:      repo,
		PromptSvc: promptSvc,
		AIClient:  openai.NewClient(apiKey),
		ImageSvc:  imgSvc,
	}
}

func (s *LocationService) GetOrEnrichLocation(ctx context.Context, inputName string) (*domain.Location, error) {
	// 1. CEK DB
	loc, err := s.Repo.FindByName(ctx, inputName)

	// Jika tidak ada error, berarti ketemu
	if err == nil && loc != nil {
		log.Printf("✅ Location found in DB: %s", loc.Name)
		return loc, nil
	}

	// Jika error-nya BUKAN karena data kosong, berarti ada masalah DB (Jangan tanya AI!)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("❌ Database Error saat mencari '%s': %v", inputName, err)
		return nil, err
	}

	// 2. JIKA BENAR-BENAR TIDAK ADA (sql.ErrNoRows) -> TANYA AI
	log.Printf("🤖 Location '%s' not found. Asking AI to enrich data...", inputName)

	meta, err := s.askAIForMetadata(ctx, inputName)
	if err != nil {
		return nil, err
	}

	// 3. SEEDING KE DB
	// Cari gambar: "Yogyakarta tourism" atau "Hotel X"
	//imageQuery := meta.Name + " tourism landmark"
	//photoURL := s.ImageSvc.SearchImage(imageQuery)
	//log.Printf("📸 [ImageService] Found photo for %s: %s", meta.Name, photoURL)

	newLoc := domain.Location{
		ID:          uuid.New().String(),
		Name:        meta.Name, // Pakai nama resmi dari AI (e.g. "Belitung Regency")
		Country:     meta.Country,
		Description: meta.Description,
		StyleTags:   meta.Styles,
		TransportHub: domain.TransportHub{
			ID:      uuid.New().String(),
			Type:    meta.HubType,
			Code:    meta.HubCode,
			Name:    meta.HubName,
			City:    meta.Name,
			Country: meta.Country,
		},
	}

	// Set Foreign Key manual di struct sebelum save (biar rapi)
	newLoc.TransportHub.LocationID = newLoc.ID

	if err := s.Repo.SaveLocation(ctx, newLoc); err != nil {
		log.Printf("⚠️ [Repo Fail] Failed to seed location to DB. Reason: %v", err)

		return &newLoc, nil
	} else {
		log.Printf("💾 Successfully seeded '%s' to DB", newLoc.Name)
	}

	return &newLoc, nil
}

func (s *LocationService) askAIForMetadata(ctx context.Context, name string) (*domain.LocationMetadataResponse, error) {
	// 1. Get System Prompt
	sysPrompt, err := s.PromptSvc.GetRenderedPrompt(ctx, "enrichment_system", nil)
	if err != nil {
		return nil, err
	}

	// 2. Get User Prompt
	data := map[string]string{"Location": name}
	userPrompt, err := s.PromptSvc.GetRenderedPrompt(ctx, "enrichment_user", data)
	if err != nil {
		return nil, err
	}

	log.Printf("🤖 [Enrichment] Sending Request for '%s'...", name)

	// 3. Call OpenAI
	resp, err := s.AIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3,
	})

	// ... (Parsing logic sama) ...
	if err != nil {
		return nil, err
	}
	content := resp.Choices[0].Message.Content
	log.Printf("📦 [Enrichment] RAW Response from AI:\n%s", content)
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")

	var meta domain.LocationMetadataResponse
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil, err
	}

	// --- LOGGING POINT 3: RESULT CHECK ---
	log.Printf("✅ [Enrichment] Parsed Struct: OfficialName='%s', Hub='%s', TagsLen=%d",
		meta.Name,
		meta.HubCode,
		len(meta.Styles),
	)

	return &meta, nil
}
