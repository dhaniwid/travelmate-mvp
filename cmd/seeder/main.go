package main

import (
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

	fmt.Println()
	log.Printf("✅ Seeding completed! Successfully processed %d/%d locations.", successCount, len(locations))
}
