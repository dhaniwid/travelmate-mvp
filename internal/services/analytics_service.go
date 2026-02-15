package services

import (
	"context"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type AnalyticsService struct {
	repo *repositories.AnalyticsRepository
}

func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// TrackEvent mencatat event baru dari user
func (s *AnalyticsService) TrackEvent(ctx context.Context, userID, eventType string, data map[string]interface{}) error {
	event := domain.AnalyticsEvent{
		UserID:    userID,
		EventType: eventType,
		EventData: data,
	}
	return s.repo.SaveEvent(ctx, event)
}

// GetImpactStats mengambil data untuk dashboard impact user
func (s *AnalyticsService) GetImpactStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	return s.repo.GetUserStats(ctx, userID)
}
