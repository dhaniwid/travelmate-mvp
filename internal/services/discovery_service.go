package services

import (
	"context"
	"encoding/json"
	"fmt"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type DiscoveryService struct {
	repo *repositories.DestinationRepository
}

func NewDiscoveryService(repo *repositories.DestinationRepository) *DiscoveryService {
	return &DiscoveryService{repo: repo}
}

// GetTrendingDestinations returns top N trending destinations
func (s *DiscoveryService) GetTrendingDestinations(ctx context.Context, limit int) ([]domain.Destination, error) {
	if limit <= 0 {
		limit = 5 // Default limit
	}
	return s.repo.GetTrending(ctx, limit)
}

// ExploreResponse structure for the explore endpoint
type ExploreResponse struct {
	Categories []string             `json:"categories"`
	Popular    []domain.Destination `json:"popular"`
	VisaFree   []domain.Destination `json:"visa_free"`
}

// GetExploreData aggregates data for the Explore page
func (s *DiscoveryService) GetExploreData(ctx context.Context) (*ExploreResponse, error) {
	// 1. Fetch Key Categories
	categories := []string{"Beach", "City", "Nature", "Culinary", "Culture"}

	// 2. Fetch All Destinations to filter in memory (or use specialized queries)
	// For MVP with small data, fetching all is fine. For scale, use specific repo methods.
	allDests, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch destinations: %w", err)
	}

	// 3. Filter Popular (Trending)
	var popular []domain.Destination
	for _, d := range allDests {
		if d.IsTrending {
			popular = append(popular, d)
		}
	}
	// Limit popular to 6
	if len(popular) > 6 {
		popular = popular[:6]
	}

	// 4. Filter Visa-Free (Tag based)
	var visaFree []domain.Destination
	for _, d := range allDests {
		// Parse tags
		var tags []string
		if len(d.Tags) > 0 {
			_ = json.Unmarshal(d.Tags, &tags)
		}

		isVisaFree := false
		for _, t := range tags {
			if t == "Visa-Free" || t == "visa-free" {
				isVisaFree = true
				break
			}
		}

		if isVisaFree {
			visaFree = append(visaFree, d)
		}
	}

	return &ExploreResponse{
		Categories: categories,
		Popular:    popular,
		VisaFree:   visaFree,
	}, nil
}
