package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"travelmate/internal/domain"

	"github.com/google/uuid"
)

// KnowledgeRepository handles all DB operations for the local_knowledge table (pgvector RAG).
type KnowledgeRepository struct {
	DB *sql.DB
}

func NewKnowledgeRepository(db *sql.DB) *KnowledgeRepository {
	return &KnowledgeRepository{DB: db}
}

// pgvectorLiteral formats a []float32 slice into a PostgreSQL vector literal string.
// e.g. []float32{0.1, 0.2, 0.3} -> "[0.1,0.2,0.3]"
// This avoids a dependency on pgvector-go; lib/pq handles the string cast natively.
func pgvectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%v", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// InsertKnowledge inserts a new local knowledge item with its pre-computed embedding.
// If knowledge.Embedding is nil/empty, the embedding column is stored as NULL.
func (r *KnowledgeRepository) InsertKnowledge(ctx context.Context, knowledge *domain.LocalKnowledge) error {
	if knowledge.ID == "" {
		knowledge.ID = uuid.New().String()
	}
	if knowledge.CreatedAt.IsZero() {
		knowledge.CreatedAt = time.Now()
	}

	var embeddingVal interface{}
	if len(knowledge.Embedding) > 0 {
		embeddingVal = pgvectorLiteral(knowledge.Embedding)
	} else {
		embeddingVal = nil // store NULL until OpenAI embedding is computed
	}

	query := `
		INSERT INTO local_knowledge (id, city, name, description, category, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::vector, $7)
	`

	_, err := r.DB.ExecContext(ctx, query,
		knowledge.ID,
		knowledge.City,
		knowledge.Name,
		knowledge.Description,
		knowledge.Category,
		embeddingVal,
		knowledge.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("knowledge_repo: failed to insert knowledge item: %w", err)
	}

	return nil
}

// SearchSimilarKnowledge performs a vector cosine-distance similarity search
// against the local_knowledge table, filtered by city.
//
// The <=> operator (pgvector cosine distance) returns values in [0, 2].
// Results are ordered ascending so the nearest (most similar) items come first.
//
// NOTE: This requires embeddings to be populated for meaningful results.
// While embeddings are NULL, this query will return 0 rows.
func (r *KnowledgeRepository) SearchSimilarKnowledge(ctx context.Context, city string, queryEmbedding []float32, limit int) ([]domain.LocalKnowledge, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("knowledge_repo: query embedding cannot be empty")
	}

	if limit <= 0 {
		limit = 5
	}

	queryVec := pgvectorLiteral(queryEmbedding)

	query := `
		SELECT id, city, name, description, category, created_at
		FROM local_knowledge
		WHERE city ILIKE $1
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> $2::vector
		LIMIT $3
	`

	rows, err := r.DB.QueryContext(ctx, query, city, queryVec, limit)
	if err != nil {
		return nil, fmt.Errorf("knowledge_repo: vector search failed: %w", err)
	}
	defer rows.Close()

	var results []domain.LocalKnowledge
	for rows.Next() {
		var k domain.LocalKnowledge
		if err := rows.Scan(
			&k.ID,
			&k.City,
			&k.Name,
			&k.Description,
			&k.Category,
			&k.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("knowledge_repo: failed to scan row: %w", err)
		}
		// NOTE: Embedding is intentionally NOT scanned here — raw vectors are
		// not returned to callers to avoid large memory allocations.
		results = append(results, k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("knowledge_repo: rows iteration error: %w", err)
	}

	return results, nil
}
