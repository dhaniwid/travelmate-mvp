package landmark

import (
	"testing"
)

func TestCheckCache_PrebundledWithoutDiskFile(t *testing.T) {
	// Simulate Railway container where LANDMARK_ASSET_DIR is an empty directory
	emptyDir := t.TempDir()
	repo := NewRepo(nil, emptyDir, "/assets/landmarks")
	svc := NewService(repo, nil)

	for _, slug := range []string{"bali", "bandung", "jakarta", "semarang", "yogyakarta"} {
		for _, variant := range []string{VariantLandscape, VariantCrispMorning, VariantMoodRain, VariantNeonNight} {
			path, hit := svc.CheckCache(slug, variant)
			if !hit {
				t.Errorf("expected cache hit for prebundled %s/%s even with empty disk dir", slug, variant)
			}
			expected := "/assets/landmarks/" + slug + "_" + variant + ".webp"
			if path != expected {
				t.Errorf("expected path %q, got %q", expected, path)
			}
		}
	}

	// Non-prebundled city on empty disk should be a cache miss
	if _, hit := svc.CheckCache("malang", VariantLandscape); hit {
		t.Errorf("expected cache miss for non-prebundled city malang on empty disk")
	}
}

func TestVariantSize_DallE3Compatible(t *testing.T) {
	if got := variantSize(VariantLandscape); got != "1792x1024" {
		t.Errorf("expected landscape size 1792x1024, got %s", got)
	}
	if got := variantSize(VariantCrispMorning); got != "1024x1024" {
		t.Errorf("expected crisp_morning size 1024x1024, got %s", got)
	}
}
