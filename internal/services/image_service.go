package services

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
)

type ImageService struct {
	APIKey string
	CX     string
}

func NewImageService(apiKey, cx string) *ImageService {
	return &ImageService{
		APIKey: apiKey,
		CX:     cx,
	}
}

func (s *ImageService) SearchImage(query string) string {
	if s.APIKey == "" || s.CX == "" {
		return getRandomPlaceholder(query)
	}

	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Add("key", s.APIKey)
	params.Add("cx", s.CX)
	params.Add("q", query)
	params.Add("searchType", "image")
	params.Add("num", "1")
	params.Add("imgSize", "large")
	params.Add("safe", "active")

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		log.Printf("⚠️ [ImageService] Request failed: %v", err)
		return getRandomPlaceholder(query)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Baca body untuk tahu alasan detail (misal: Quota Exceeded)
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [ImageService] Google API Error %d: %s", resp.StatusCode, string(body))
		return getRandomPlaceholder(query)
	}

	var result struct {
		Items []struct {
			Link string `json:"link"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("⚠️ [ImageService] Decode failed: %v", err)
		return getRandomPlaceholder(query)
	}

	if len(result.Items) > 0 {
		return result.Items[0].Link
	}

	return getRandomPlaceholder(query)
}

func getRandomPlaceholder(_ string) string {
	placeholders := []string{
		"https://images.unsplash.com/photo-1507525428034-b723cf961d3e?w=800&q=80",
		"https://images.unsplash.com/photo-1476514525535-07fb3b4ae5f1?w=800&q=80",
		"https://images.unsplash.com/photo-1469854523086-cc02fe5d8800?w=800&q=80",
		"https://images.unsplash.com/photo-1501785888041-af3ef285b470?w=800&q=80",
		"https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800&q=80",
		"https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800&q=80",
		"https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800&q=80",
	}

	// rand.Seed sudah deprecated sejak Go 1.20, sekarang otomatis
	return placeholders[rand.Intn(len(placeholders))]
}
