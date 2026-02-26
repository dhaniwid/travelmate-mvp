package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"travelmate/internal/domain"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

// KnowledgeIngester defines the repository interface the handler depends on.
type KnowledgeIngester interface {
	InsertKnowledge(ctx context.Context, knowledge *domain.LocalKnowledge) error
}

// KnowledgeHandler handles admin ingestion of local knowledge for RAG.
type KnowledgeHandler struct {
	repo         KnowledgeIngester
	openaiClient *openai.Client
}

func NewKnowledgeHandler(repo KnowledgeIngester, openaiClient *openai.Client) *KnowledgeHandler {
	return &KnowledgeHandler{
		repo:         repo,
		openaiClient: openaiClient,
	}
}

// IngestKnowledgeRequest is the payload accepted by the admin endpoint.
type IngestKnowledgeRequest struct {
	City        string `json:"city" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Category    string `json:"category" binding:"required"`
}

// buildEmbeddingInput constructs a rich text representation of the knowledge item
// for higher-quality embeddings. Combining multiple fields gives the model better semantic context.
func buildEmbeddingInput(req IngestKnowledgeRequest) string {
	return fmt.Sprintf(
		"City: %s, Name: %s, Category: %s. Context: %s",
		req.City, req.Name, req.Category, req.Description,
	)
}

// IngestKnowledge handles POST /api/v1/admin/knowledge
// Protected by AdminAuthMiddleware.
//
// Flow:
//  1. Parse & validate request body.
//  2. Call OpenAI text-embedding-3-small to compute a 1536-dim vector.
//  3. Insert record + embedding into local_knowledge (pgvector).
//
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body IngestKnowledgeRequest true "Knowledge item to ingest"
// @Success 201 {object} map[string]string
// @Router /api/v1/admin/knowledge [post]
func (h *KnowledgeHandler) IngestKnowledge(c *gin.Context) {
	var req IngestKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// ── Step 1: Build rich embedding input text ──────────────────────────────
	inputText := buildEmbeddingInput(req)
	log.Printf("🧠 [Knowledge] Generating embedding for: %q", inputText)

	// ── Step 2: Call OpenAI text-embedding-3-small ───────────────────────────
	embResp, err := h.openaiClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{inputText},
		Model: openai.SmallEmbedding3, // text-embedding-3-small — 1536 dims
	})
	if err != nil {
		log.Printf("❌ [Knowledge] OpenAI embedding failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "embedding_error",
			"message": "Failed to generate embedding from OpenAI",
		})
		return
	}
	if len(embResp.Data) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "embedding_empty",
			"message": "OpenAI returned no embedding data",
		})
		return
	}

	// ── Step 3: Build knowledge item with embedding ──────────────────────────
	knowledge := &domain.LocalKnowledge{
		City:        req.City,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Embedding:   embResp.Data[0].Embedding,
	}

	log.Printf("🧠 [Knowledge] Embedding generated (%d dims). Saving to DB...", len(knowledge.Embedding))

	// ── Step 4: Persist to local_knowledge table (pgvector) ─────────────────
	if err := h.repo.InsertKnowledge(ctx, knowledge); err != nil {
		log.Printf("❌ [Knowledge] DB insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": "Failed to save knowledge item to database",
		})
		return
	}

	log.Printf("✅ [Knowledge] Saved item %q in %s [%s] ID=%s", knowledge.Name, knowledge.City, knowledge.Category, knowledge.ID)

	c.JSON(http.StatusCreated, gin.H{
		"id":       knowledge.ID,
		"message":  "Knowledge ingested and embedded successfully",
		"city":     knowledge.City,
		"name":     knowledge.Name,
		"category": knowledge.Category,
		"dims":     len(knowledge.Embedding),
	})
}
