package landmark

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/sync/singleflight"
)

// ── Variant constants ────────────────────────────────────────────────────────

const (
	VariantLandscape    = "landscape"
	VariantCrispMorning = "crisp_morning"
	VariantMoodRain     = "mood_rain"
	VariantNeonNight    = "neon_night"
)

var validVariants = map[string]bool{
	VariantLandscape:    true,
	VariantCrispMorning: true,
	VariantMoodRain:     true,
	VariantNeonNight:    true,
}

// ── Prompt templates (source: miru-docs/design/landmark-prompt-templates.md) ─

type tmplData struct {
	City         string
	CityUpper    string
	Landmark     string
	LandmarkDesc string
	GeoContext   string
}

var rawTemplates = map[string]string{
	VariantLandscape: `A hyper-realistic 3D miniature landscape render of {{.City}} Indonesia, ` +
		`eye-level perspective at slight elevation. Centerpiece is {{.Landmark}}: ` +
		`{{.LandmarkDesc}}. Vibrant teal-green color on all window frames, roof ` +
		`edges, and architectural details. Lush manicured green lawn in foreground, ` +
		`tropical palm trees framing both sides, vivid orange flower beds flanking ` +
		`the entrance path. {{.GeoContext}}. Soft warm afternoon light, rich ` +
		`saturated colors, matte clay miniature texture throughout, seamless ground ` +
		`extending to edges with no circular platform or base. Wide cinematic ` +
		`composition, photorealistic, 8k.`,

	VariantCrispMorning: `A hyper-realistic 3D render of a rounded miniature display platform with ` +
		`pure white matte base, showcasing a detailed 3D diorama of {{.City}} ` +
		`Indonesia. The centerpiece is {{.Landmark}}: {{.LandmarkDesc}}. ` +
		`Surrounded by lush tropical miniature greenery and palm trees. Soft cool ` +
		`morning light from upper left, long gentle shadows, fresh dewy atmosphere, ` +
		`vibrant teal accent color on windows and platform details, vivid orange ` +
		`flower beds. Bold embossed 3D {{.CityUpper}} lettering in teal on the ` +
		`front edge of the white platform base. Top-down 3/4 perspective, ` +
		`photorealistic clay render, 8k.`,

	VariantMoodRain: `A hyper-realistic 3D render of a rounded miniature display platform with ` +
		`pure white matte base, showcasing a detailed 3D diorama of {{.City}} ` +
		`Indonesia. The centerpiece is {{.Landmark}}: {{.LandmarkDesc}}. ` +
		`Surrounded by lush tropical miniature greenery and palm trees with rain ` +
		`droplets. Overcast diffused lighting, desaturated muted tones, wet ` +
		`reflective surfaces on the platform, grey-blue atmosphere, teal accent ` +
		`color on windows and platform details, dark orange flower beds. Bold ` +
		`embossed 3D {{.CityUpper}} lettering in teal on the front edge of the ` +
		`white platform base. Top-down 3/4 perspective, photorealistic clay render, 8k.`,

	VariantNeonNight: `A hyper-realistic 3D render of a rounded miniature display platform with ` +
		`pure white matte base, showcasing a detailed 3D diorama of {{.City}} ` +
		`Indonesia at night. The centerpiece is {{.Landmark}}: {{.LandmarkDesc}}. ` +
		`Surrounded by lush tropical miniature greenery and palm trees with warm ` +
		`artificial lighting. Dramatic night scene, deep blue-black sky above, warm ` +
		`amber and teal neon glow from building windows and platform base, high ` +
		`contrast shadows, teal accent strongly illuminated, glowing orange flower ` +
		`beds. Bold embossed 3D {{.CityUpper}} lettering in glowing teal on the ` +
		`front edge of the white platform base. Top-down 3/4 perspective, ` +
		`photorealistic clay render, 8k.`,
}

var compiledTemplates map[string]*template.Template

func init() {
	compiledTemplates = make(map[string]*template.Template, len(rawTemplates))
	for variant, raw := range rawTemplates {
		compiledTemplates[variant] = template.Must(template.New(variant).Parse(raw))
	}
}

// variantSuffix holds per-city, per-variant prompt additions that correct
// known AI hallucinations (e.g. semarang landscape generating domes instead of spires).
var variantSuffix = map[string]map[string]string{
	"semarang": {
		VariantLandscape: "The building must be white/cream colored with pointed gothic spires, not domes, not orange or terracotta.",
	},
}

// variantSize maps variant name to the image size string for the OpenAI API.
func variantSize(variant string) string {
	if variant == VariantLandscape {
		return "1536x1024"
	}
	return "1024x1024"
}

// ── Service ──────────────────────────────────────────────────────────────────

// Service orchestrates landmark image generation and caching.
type Service struct {
	repo   LandmarkRepository
	ai     *openai.Client
	flight singleflight.Group
}

func NewService(repo LandmarkRepository, aiClient *openai.Client) *Service {
	return &Service{repo: repo, ai: aiClient}
}

// CheckCache returns (publicPath, true) if the image is already cached.
func (s *Service) CheckCache(slug, variant string) (string, bool) {
	if s.repo.Exists(slug, variant) {
		return s.repo.PublicPath(slug, variant), true
	}
	return "", false
}

// GetOrGenerate returns the public path for slug+variant, generating the image
// if not cached. Concurrent calls for the same key share a single in-flight
// generation request (singleflight).
func (s *Service) GetOrGenerate(ctx context.Context, slug, variant string) (string, error) {
	if path, hit := s.CheckCache(slug, variant); hit {
		return path, nil
	}

	key := slug + ":" + variant
	result, err, _ := s.flight.Do(key, func() (interface{}, error) {
		if path, hit := s.CheckCache(slug, variant); hit {
			return path, nil
		}
		return s.generate(ctx, slug, variant)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// ListSlugs returns all city slugs from the DB for use by SeedLandmarks.
func (s *Service) ListSlugs(ctx context.Context) ([]string, error) {
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	slugs := make([]string, len(configs))
	for i, c := range configs {
		slugs[i] = c.Slug
	}
	return slugs, nil
}

// ValidateSlug checks whether a slug exists in the DB.
func (s *Service) ValidateSlug(ctx context.Context, slug string) (bool, error) {
	cfg, err := s.repo.GetConfig(ctx, slug)
	if err != nil {
		return false, err
	}
	return cfg != nil, nil
}

func (s *Service) generate(ctx context.Context, slug, variant string) (string, error) {
	if !validVariants[variant] {
		return "", fmt.Errorf("unknown variant: %s", variant)
	}

	cfg, err := s.repo.GetConfig(ctx, slug)
	if err != nil {
		return "", fmt.Errorf("load city config: %w", err)
	}
	if cfg == nil {
		return "", fmt.Errorf("unknown city slug: %s", slug)
	}

	prompt, err := renderPrompt(variant, *cfg)
	if err != nil {
		return "", fmt.Errorf("render prompt: %w", err)
	}

	log.Printf("[landmark] generating %s/%s (size=%s)", slug, variant, variantSize(variant))

	// gpt-image-1 does not accept response_format — it always returns b64_json.
	resp, err := s.ai.CreateImage(ctx, openai.ImageRequest{
		Model:   "gpt-image-1",
		Prompt:  prompt,
		N:       1,
		Size:    variantSize(variant),
		Quality: "high",
	})
	if err != nil {
		return "", fmt.Errorf("openai CreateImage: %w", err)
	}
	if len(resp.Data) == 0 || resp.Data[0].B64JSON == "" {
		return "", fmt.Errorf("openai returned empty image data")
	}

	if err := s.repo.SaveB64(slug, variant, resp.Data[0].B64JSON); err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}

	log.Printf("[landmark] saved %s/%s → %s", slug, variant, s.repo.PublicPath(slug, variant))
	return s.repo.PublicPath(slug, variant), nil
}

func renderPrompt(variant string, cfg DBCityConfig) (string, error) {
	tmpl, ok := compiledTemplates[variant]
	if !ok {
		return "", fmt.Errorf("no template for variant %s", variant)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData{
		City:         cfg.CityName,
		CityUpper:    strings.ToUpper(cfg.CityName),
		Landmark:     cfg.LandmarkName,
		LandmarkDesc: cfg.LandmarkDesc,
		GeoContext:   cfg.GeoContext,
	}); err != nil {
		return "", err
	}
	prompt := buf.String()
	if cityMap, ok := variantSuffix[cfg.Slug]; ok {
		if suffix, ok := cityMap[variant]; ok && suffix != "" {
			prompt += " " + suffix
		}
	}
	return prompt, nil
}

