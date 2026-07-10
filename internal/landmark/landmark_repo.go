package landmark

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
)

// LandmarkRepository is the interface consumed by Service.
type LandmarkRepository interface {
	GetConfig(ctx context.Context, slug string) (*DBCityConfig, error)
	ListConfigs(ctx context.Context) ([]DBCityConfig, error)
	Exists(slug, variant string) bool
	SaveB64(slug, variant, b64data string) error
	PublicPath(slug, variant string) string
}

// DBCityConfig holds per-city data loaded from the landmark_configs table.
type DBCityConfig struct {
	Slug         string
	CityName     string
	LandmarkName string
	LandmarkDesc string
	GeoContext   string
}

// Repo handles DB queries and filesystem read/write for cached landmark images.
type Repo struct {
	db         *sql.DB
	baseDir    string
	publicBase string
}

func NewRepo(db *sql.DB, baseDir, publicBase string) *Repo {
	return &Repo{db: db, baseDir: baseDir, publicBase: publicBase}
}

// GetConfig loads a single city config from the DB by slug.
// Returns (nil, nil) when the slug is not found.
func (r *Repo) GetConfig(ctx context.Context, slug string) (*DBCityConfig, error) {
	var c DBCityConfig
	err := r.db.QueryRowContext(ctx, `
		SELECT slug, city_name, landmark_name, landmark_desc, COALESCE(geo_context, '')
		FROM landmark_configs
		WHERE slug = $1
	`, slug).Scan(&c.Slug, &c.CityName, &c.LandmarkName, &c.LandmarkDesc, &c.GeoContext)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query landmark config: %w", err)
	}
	return &c, nil
}

// ListConfigs returns all city configs from the DB.
func (r *Repo) ListConfigs(ctx context.Context) ([]DBCityConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT slug, city_name, landmark_name, landmark_desc, COALESCE(geo_context, '')
		FROM landmark_configs
		ORDER BY slug
	`)
	if err != nil {
		return nil, fmt.Errorf("list landmark configs: %w", err)
	}
	defer rows.Close()

	var configs []DBCityConfig
	for rows.Next() {
		var c DBCityConfig
		if err := rows.Scan(&c.Slug, &c.CityName, &c.LandmarkName, &c.LandmarkDesc, &c.GeoContext); err != nil {
			return nil, fmt.Errorf("scan landmark config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *Repo) filePath(slug, variant string) string {
	return filepath.Join(r.baseDir, fmt.Sprintf("%s_%s.webp", slug, variant))
}

// Exists returns true if a cached file is already present for this slug+variant.
func (r *Repo) Exists(slug, variant string) bool {
	_, err := os.Stat(r.filePath(slug, variant))
	return err == nil
}

// SaveB64 decodes a base64-encoded PNG from the OpenAI b64_json response,
// re-encodes it as WebP (quality 90), and persists the result to disk.
func (r *Repo) SaveB64(slug, variant, b64data string) error {
	if err := os.MkdirAll(r.baseDir, 0o755); err != nil {
		return fmt.Errorf("landmark mkdir %s: %w", r.baseDir, err)
	}

	raw, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return fmt.Errorf("landmark b64 decode: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("landmark png decode: %w", err)
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: 90}); err != nil {
		return fmt.Errorf("landmark webp encode: %w", err)
	}

	path := r.filePath(slug, variant)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("landmark write %s: %w", path, err)
	}
	return nil
}

// PublicPath returns the URL path the frontend uses to reference the cached image.
func (r *Repo) PublicPath(slug, variant string) string {
	return fmt.Sprintf("%s/%s_%s.webp", r.publicBase, slug, variant)
}
