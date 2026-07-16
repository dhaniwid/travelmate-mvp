package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"travelmate/internal/config"
	"travelmate/internal/db"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"

	_ "github.com/lib/pq"
	openai "github.com/sashabaranov/go-openai"
)

type KnowledgeSeed struct {
	City        string `json:"city"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	database := db.Connect(cfg.DBUrl)
	defer database.Close()
	log.Println("🔌 Database connected.")

	if cfg.OpenAIKey == "" {
		log.Fatal("❌ OPENAI_API_KEY is not set")
	}
	aiClient := openai.NewClient(cfg.OpenAIKey)
	repo := repositories.NewKnowledgeRepository(database)

	data, err := os.ReadFile("seeds/local_knowledge.json")
	if err != nil {
		log.Fatalf("❌ Cannot read seeds/local_knowledge.json: %v", err)
	}

	var items []KnowledgeSeed
	if err := json.Unmarshal(data, &items); err != nil {
		log.Fatalf("❌ Cannot parse JSON: %v", err)
	}
	log.Printf("🌱 Seeding %d local knowledge items...", len(items))

	ok, fail := 0, 0
	for _, item := range items {
		// Build ingestion text — mirrors the query format used in fetchLocalKnowledge
		ingestionText := fmt.Sprintf("City: %s | %s | %s: %s", item.City, item.Category, item.Name, item.Description)

		embResp, err := aiClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Input: []string{ingestionText},
			Model: openai.SmallEmbedding3,
		})
		if err != nil {
			log.Printf("⚠️  [%s] Embedding failed: %v", item.Name, err)
			fail++
			continue
		}

		embedding := embResp.Data[0].Embedding

		k := &domain.LocalKnowledge{
			City:        item.City,
			Name:        item.Name,
			Description: item.Description,
			Category:    item.Category,
			Embedding:   embedding,
		}

		if err := repo.InsertKnowledge(ctx, k); err != nil {
			log.Printf("⚠️  [%s] DB insert failed: %v", item.Name, err)
			fail++
			continue
		}

		fmt.Printf("✅ %s — %s\n", item.City, item.Name)
		ok++
	}

	fmt.Printf("\n🏁 Done: %d inserted, %d failed.\n", ok, fail)
}
