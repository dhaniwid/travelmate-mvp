package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings" // <--- Import strings

	"travelmate/internal/config"
	"travelmate/internal/db"

	_ "github.com/lib/pq"
)

type LocationSeed struct {
	Name        string `json:"name"`
	Country     string `json:"country"`
	Description string `json:"description"`
	StyleTags   string `json:"style_tags"` // Di JSON: "Tag1,Tag2"
	HubType     string `json:"hub_type"`
	HubCode     string `json:"hub_code"`
	HubName     string `json:"hub_name"`
	ImageURL    string `json:"image_url"`
}

func main() {
	// 1. Config
	cfg := config.LoadConfig()

	// 2. Database
	database := db.Connect(cfg.DBUrl)
	defer database.Close()

	log.Println("🔌 Database connected for seeding...")

	// 3. Baca File JSON
	fileContent, err := os.ReadFile("seeds/locations.json")
	if err != nil {
		log.Fatalf("❌ Error reading seeds/locations.json: %v", err)
	}

	var locations []LocationSeed
	if err := json.Unmarshal(fileContent, &locations); err != nil {
		log.Fatalf("❌ Error parsing JSON: %v", err)
	}

	log.Printf("🌱 Found %d locations to seed...", len(locations))

	// 4. Seeding Loop
	successCount := 0
	for _, loc := range locations {
		// --- FIX: KONVERSI STYLE_TAGS STRING KE JSON ARRAY ---
		// Input: "Nature,Culture,Beach"
		// Output: ["Nature", "Culture", "Beach"] (sebagai JSON String)

		var tagsJSON []byte
		if loc.StyleTags != "" {
			// 1. Split string berdasarkan koma
			tagsList := strings.Split(loc.StyleTags, ",")

			// 2. Bersihkan spasi di setiap tag
			for i := range tagsList {
				tagsList[i] = strings.TrimSpace(tagsList[i])
			}

			// 3. Ubah jadi JSON Byte Array
			tagsJSON, _ = json.Marshal(tagsList)
		} else {
			// Jika kosong, set array kosong JSON
			tagsJSON = []byte("[]")
		}
		// -----------------------------------------------------

		query := `
			INSERT INTO locations (
				id, name, country, description, style_tags, 
				hub_type, hub_code, hub_name, image_url, last_updated
			)
			VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW()
			)
			ON CONFLICT (name) DO UPDATE SET 
				country = EXCLUDED.country,
				description = EXCLUDED.description,
				style_tags = EXCLUDED.style_tags,
				hub_type = EXCLUDED.hub_type,
				hub_code = EXCLUDED.hub_code,
				hub_name = EXCLUDED.hub_name,
				image_url = EXCLUDED.image_url,
				last_updated = NOW();
		`

		_, err := database.Exec(query,
			loc.Name,
			loc.Country,
			loc.Description,
			string(tagsJSON), // <--- Kirim sebagai JSON String yang valid
			loc.HubType,
			loc.HubCode,
			loc.HubName,
			loc.ImageURL,
		)

		if err != nil {
			log.Printf("⚠️ Failed to seed '%s': %v", loc.Name, err)
		} else {
			successCount++
			fmt.Print(".")
		}
	}

	// 5. Seed Destinations (New Task)
	seedDestinations(database)

	fmt.Println()
	log.Printf("✅ Seeding completed! Successfully processed %d/%d locations.", successCount, len(locations))
}

type DestinationSeed struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	IsTrending  bool     `json:"is_trending"`
}

func seedDestinations(db *sql.DB) {
	log.Println("\n🌴 Seeding Destinations (M-120)...")

	destinations := []DestinationSeed{
		{
			Name:        "Kyoto",
			Country:     "Japan",
			Description: "Ancient temples, traditional teahouses, and sublime gardens.",
			ImageURL:    "https://images.unsplash.com/photo-1493976040374-85c8e12f0c0e", // Provided Asset
			Category:    "City",
			Tags:        []string{"City", "Culture", "Spring"},
			IsTrending:  true,
		},
		{
			Name:        "Nusa Penida",
			Country:     "Indonesia",
			Description: "Dramatic cliffs and pristine beaches off the coast of Bali.",
			ImageURL:    "https://images.unsplash.com/photo-1537996194471-e657df975ab4", // Provided Asset
			Category:    "Beach",
			Tags:        []string{"Beach", "Nature", "Visa-Free"},
			IsTrending:  true,
		},
		{
			Name:        "Interlaken",
			Country:     "Switzerland",
			Description: "Adventure capital nestled between two lakes and the Alps.",
			ImageURL:    "https://images.unsplash.com/photo-1530122037265-a5f1f91d3b99", // Provided Asset
			Category:    "Nature",
			Tags:        []string{"Nature", "Mountain"},
			IsTrending:  true,
		},
		{
			Name:        "Seoul",
			Country:     "South Korea",
			Description: "A high-tech city with deep traditional roots and amazing food.",
			ImageURL:    "https://images.unsplash.com/photo-1633912891963-31f03403248e", // Provided Asset
			Category:    "City",
			Tags:        []string{"City", "Culinary", "Nightlife"},
			IsTrending:  false,
		},
		{
			Name:        "Paris",
			Country:     "France",
			Description: "The city of love, art, and exquisite cuisine.",
			ImageURL:    "https://images.unsplash.com/photo-1502602898657-3e91760cbb34", // Provided Asset
			Category:    "City",
			Tags:        []string{"City", "Romance"},
			IsTrending:  false,
		},
		{
			Name:        "New York",
			Country:     "USA",
			Description: "The city that never sleeps, a melting pot of cultures.",
			ImageURL:    "https://images.unsplash.com/photo-1496442226666-8d4d0e62e6e9", // Provided Asset
			Category:    "City",
			Tags:        []string{"City", "Urban"},
			IsTrending:  false,
		},
	}

	for _, dest := range destinations {
		tagsJSON, _ := json.Marshal(dest.Tags)

		query := `
			INSERT INTO destinations (
				id, name, country, description, image_url, category, tags, is_trending
			)
			VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7
			)
			ON CONFLICT (id) DO NOTHING; -- UUID conflict unlikely, but safety first
		`
		// Note: We don't have a unique constraint on Name on the new table yet (unless we added one?)
		// To allow re-seeding without duplicates if we run this multiple times, we might want to check existence strictly.
		// For now, let's use a check against Name to prevent dupes if running multiple times.
		// Actually, let's modify the query to check existence by name.
		checkQuery := `SELECT id FROM destinations WHERE name = $1`
		var existingID string
		err := db.QueryRow(checkQuery, dest.Name).Scan(&existingID)
		if err == nil {
			// Update exisiting
			updateQuery := `
				UPDATE destinations SET 
					country = $2, description = $3, image_url = $4, category = $5, tags = $6, is_trending = $7
				WHERE name = $1
			`
			_, err = db.Exec(updateQuery, dest.Name, dest.Country, dest.Description, dest.ImageURL, dest.Category, tagsJSON, dest.IsTrending)
			if err != nil {
				log.Printf("⚠️ Failed to update destination '%s': %v", dest.Name, err)
			} else {
				fmt.Print("U")
			}
		} else {
			// Insert new
			_, err = db.Exec(query, dest.Name, dest.Country, dest.Description, dest.ImageURL, dest.Category, tagsJSON, dest.IsTrending)
			if err != nil {
				log.Printf("⚠️ Failed to seed destination '%s': %v", dest.Name, err)
			} else {
				fmt.Print("+")
			}
		}
	}
	log.Println("\nDone seeding destinations.")
}
