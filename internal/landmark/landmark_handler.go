package landmark

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler exposes HTTP endpoints for the landmark domain.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetLandmark handles GET /api/v1/landmarks/:slug/:variant
//
// Cache hit  → 200 { "path": "...", "cached": true }
// Cache miss → generates synchronously (may take 40–55s), then 200 { "path": "...", "cached": false }
// Unknown slug/variant → 400
// Generation failure   → 500
func (h *Handler) GetLandmark(c *gin.Context) {
	slug := c.Param("slug")
	variant := c.Param("variant")

	if !validVariants[variant] {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_variant", "message": "unknown variant"})
		return
	}

	// Cache check first — avoids DB round-trip on the hot path.
	if path, hit := h.svc.CheckCache(slug, variant); hit {
		c.JSON(http.StatusOK, gin.H{"path": path, "cached": true})
		return
	}

	ok, err := h.svc.ValidateSlug(c.Request.Context(), slug)
	if err != nil {
		log.Printf("❌ [LANDMARK GET] validate slug %s: %v", slug, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal_error", "message": "Gagal memvalidasi slug"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_slug", "message": "unknown city slug"})
		return
	}

	path, err := h.svc.GetOrGenerate(c.Request.Context(), slug, variant)
	if err != nil {
		log.Printf("❌ [LANDMARK GET] %s/%s: %v", slug, variant, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "generate_failed", "message": "Gagal memuat gambar landmark, coba lagi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": path, "cached": false})
}

// SeedLandmarks handles POST /api/v1/admin/landmarks/seed
//
// Optional JSON body: { "slugs": ["bandung","bali"], "variants": ["landscape"] }
// Defaults to all slugs from DB × all variants when body is absent.
//
// Runs all generations sequentially to avoid hammering the OpenAI API;
// singleflight still protects against concurrent seed calls.
func (h *Handler) SeedLandmarks(c *gin.Context) {
	var body struct {
		Slugs    []string `json:"slugs"`
		Variants []string `json:"variants"`
	}
	_ = c.ShouldBindJSON(&body)

	slugs := body.Slugs
	if len(slugs) == 0 {
		var err error
		slugs, err = h.svc.ListSlugs(c.Request.Context())
		if err != nil {
			log.Printf("❌ [LANDMARK SEED] list slugs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": "internal_error", "message": "Gagal memuat daftar kota"})
			return
		}
	}

	variants := body.Variants
	if len(variants) == 0 {
		variants = []string{VariantLandscape, VariantCrispMorning, VariantMoodRain, VariantNeonNight}
	}

	type result struct {
		Slug    string `json:"slug"`
		Variant string `json:"variant"`
		Path    string `json:"path,omitempty"`
		Cached  bool   `json:"cached"`
		Error   string `json:"error,omitempty"`
	}

	var results []result

	for _, slug := range slugs {
		slugOK, slugErr := h.svc.ValidateSlug(c.Request.Context(), slug)
		if slugErr != nil {
			log.Printf("❌ [LANDMARK SEED] validate slug %s: %v", slug, slugErr)
			results = append(results, result{Slug: slug, Error: "internal_error"})
			continue
		}
		if !slugOK {
			results = append(results, result{Slug: slug, Error: "invalid_slug"})
			continue
		}
		for _, variant := range variants {
			if !validVariants[variant] {
				results = append(results, result{Slug: slug, Variant: variant, Error: "invalid_variant"})
				continue
			}

			if path, hit := h.svc.CheckCache(slug, variant); hit {
				results = append(results, result{Slug: slug, Variant: variant, Path: path, Cached: true})
				continue
			}

			path, err := h.svc.GetOrGenerate(c.Request.Context(), slug, variant)
			if err != nil {
				log.Printf("❌ [LANDMARK SEED] %s/%s: %v", slug, variant, err)
				results = append(results, result{Slug: slug, Variant: variant, Error: "generate_failed"})
				continue
			}
			results = append(results, result{Slug: slug, Variant: variant, Path: path, Cached: false})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
