package services

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"sync"
)

// Kita pakai in-memory caching sederhana agar tidak hit DB setiap request
type PromptService struct {
	DB    *sql.DB
	cache map[string]string // Key -> TemplateText
	mutex sync.RWMutex
}

func NewPromptService(db *sql.DB) *PromptService {
	return &PromptService{
		DB:    db,
		cache: make(map[string]string),
	}
}

// GetRenderedPrompt mengambil template aktif, lalu mengisi variabelnya
func (s *PromptService) GetRenderedPrompt(ctx context.Context, key string, data interface{}) (string, error) {
	// 1. Cek Cache dulu
	tmplText, found := s.getFromCache(key)
	if !found {
		// 2. Jika tidak ada, ambil dari DB
		var err error
		tmplText, err = s.fetchFromDB(ctx, key)
		if err != nil {
			return "", err
		}
		// Simpan ke cache
		s.addToCache(key, tmplText)
	}

	// 3. Render Template (Replace {{.Variable}} dengan value)
	tmpl, err := template.New(key).Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", key, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", key, err)
	}

	return buf.String(), nil
}

func (s *PromptService) fetchFromDB(ctx context.Context, key string) (string, error) {
	query := `SELECT template_text FROM system_prompts WHERE key = $1 AND is_active = true LIMIT 1`
	var text string
	err := s.DB.QueryRowContext(ctx, query, key).Scan(&text)
	if err != nil {
		return "", fmt.Errorf("prompt key '%s' not found or no active version", key)
	}
	log.Printf("📝 Loaded prompt '%s' from DB", key)
	return text, nil
}

func (s *PromptService) getFromCache(key string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	val, ok := s.cache[key]
	return val, ok
}

func (s *PromptService) addToCache(key, val string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cache[key] = val
}

// Method untuk clear cache (dipanggil jika admin update prompt di DB)
func (s *PromptService) RefreshCache() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cache = make(map[string]string)
	log.Println("🧹 Prompt cache cleared")
}
