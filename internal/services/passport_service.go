package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type PassportService struct {
	Repo *repositories.PassportRepository
}

func NewPassportService(repo *repositories.PassportRepository) *PassportService {
	return &PassportService{Repo: repo}
}

// GetUserStamps returns all passport stamps for a user.
func (s *PassportService) GetUserStamps(ctx context.Context, userID string) ([]domain.PassportStamp, error) {
	stamps, err := s.Repo.GetUserStamps(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stamps == nil {
		return []domain.PassportStamp{}, nil
	}
	return stamps, nil
}

// CheckStamp returns existing stamps for a user+city (all moods).
func (s *PassportService) CheckStamp(ctx context.Context, userID, citySlug string) ([]domain.PassportStamp, error) {
	stamps, err := s.Repo.GetStampsByCity(ctx, userID, citySlug)
	if err != nil {
		return nil, err
	}
	if stamps == nil {
		return []domain.PassportStamp{}, nil
	}
	return stamps, nil
}

// ClaimStamp creates or updates a stamp for user+city+mood.
func (s *PassportService) ClaimStamp(ctx context.Context, userID, tripID, city, citySlug string) (*domain.PassportStamp, error) {
	now := time.Now()
	mood := s.detectMood(now, "")
	serial := s.generateSerial(citySlug, now, mood)
	imageURL := s.resolveImageURL(citySlug, mood)
	rotation := (rand.Float64()*14 - 7) // -7 to +7 degrees

	stamp := domain.PassportStamp{
		UserID:   userID,
		City:     city,
		CitySlug: citySlug,
		Date:     now,
		Serial:   serial,
		Mood:     mood,
		ImageURL: imageURL,
		Rotation: rotation,
		TripID:   tripID,
	}

	return s.Repo.UpsertStamp(ctx, stamp)
}

// detectMood returns mood based on hour (and optionally weather string).
// rain/cloudy → rain, 05–11 → morning, 18–04 → night
func (s *PassportService) detectMood(claimedAt time.Time, weather string) string {
	w := strings.ToLower(weather)
	if strings.Contains(w, "rain") || strings.Contains(w, "cloud") || strings.Contains(w, "drizzle") {
		return "rain"
	}
	hour := claimedAt.Hour()
	if hour >= 5 && hour < 18 {
		return "morning"
	}
	return "night"
}

// generateSerial produces a stamp serial in format [IATA]-[DDMM]-[M|R|N].
// e.g. CGK-0613-M
func (s *PassportService) generateSerial(citySlug string, claimedAt time.Time, mood string) string {
	iata := s.citySlugToIATA(citySlug)
	ddmm := claimedAt.Format("0201") // day + month
	moodCode := map[string]string{"morning": "M", "rain": "R", "night": "N"}[mood]
	return fmt.Sprintf("%s-%s-%s", iata, ddmm, moodCode)
}

// resolveImageURL returns a placeholder image URL for the stamp artwork.
// In production this would point to pre-generated stamp artwork per city+mood.
func (s *PassportService) resolveImageURL(citySlug, mood string) string {
	return fmt.Sprintf("/stamps/%s/%s.png", citySlug, mood)
}

// citySlugToIATA maps common city slugs to their main airport IATA code.
var iataMap = map[string]string{
	"jakarta":      "CGK",
	"bali":         "DPS",
	"denpasar":     "DPS",
	"yogyakarta":   "JOG",
	"surabaya":     "SUB",
	"bandung":      "BDO",
	"medan":        "KNO",
	"makassar":     "UPG",
	"lombok":       "LOP",
	"manado":       "MDC",
	"labuan_bajo":  "LBJ",
	"palembang":    "PLM",
	"semarang":     "SRG",
	"singapore":    "SIN",
	"kuala_lumpur": "KUL",
	"bangkok":      "BKK",
	"tokyo":        "TYO",
	"osaka":        "KIX",
	"paris":        "CDG",
	"london":       "LHR",
	"amsterdam":    "AMS",
	"barcelona":    "BCN",
	"rome":         "FCO",
	"dubai":        "DXB",
	"sydney":       "SYD",
	"new_york":     "JFK",
}

func (s *PassportService) citySlugToIATA(citySlug string) string {
	slug := strings.ToLower(citySlug)
	if code, ok := iataMap[slug]; ok {
		return code
	}
	// Fallback: first 3 chars of slug uppercased
	clean := strings.NewReplacer("-", "", "_", "").Replace(slug)
	if len(clean) >= 3 {
		return strings.ToUpper(clean[:3])
	}
	return strings.ToUpper(clean)
}
