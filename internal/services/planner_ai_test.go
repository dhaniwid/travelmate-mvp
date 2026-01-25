package services_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"travelmate/internal/domain"
	"travelmate/internal/services"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func BenchmarkAIPlanner(b *testing.B) {
	// 1. Setup Environment
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Printf("Warning: .env file not found, using system env")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	dbURL := os.Getenv("DATABASE_URL")

	if apiKey == "" || dbURL == "" {
		b.Skip("Skipping benchmark: OPENAI_API_KEY or DATABASE_URL not set in .env")
	}

	// 2. Inisialisasi Database Real
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		b.Fatalf("❌ Gagal koneksi database: %v", err)
	}
	defer db.Close()

	// Cek koneksi
	if err := db.Ping(); err != nil {
		b.Fatalf("❌ Database tidak merespon: %v", err)
	}

	// 3. Inisialisasi Services
	promptSvc := services.NewPromptService(db)
	planner := services.NewAIPlanner(apiKey, promptSvc)

	// Mock data untuk testing
	mockTrip := domain.Trip{
		ID:          "bench-123",
		Destination: "Lombok",
		Origin:      "Jakarta",
		TripDays:    3,
		Style:       "Adventure",
		Budget:      5000000,
	}

	ctx := context.Background()

	// Reset timer agar waktu setup database tidak dihitung dalam benchmark
	b.ResetTimer()

	// --- BENCHMARK 1: ITINERARY ONLY ---
	b.Run("Task-Itinerary-Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := planner.GenerateOnlyItinerary(ctx, mockTrip)
			if err != nil {
				b.Errorf("❌ Itinerary Task Failed: %v", err)
			}
		}
	})

	// --- BENCHMARK 2: LOGISTICS ONLY ---
	b.Run("Task-Logistics-Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := planner.GenerateTransportAndStay(ctx, mockTrip)
			if err != nil {
				b.Errorf("❌ Logistics Task Failed: %v", err)
			}
		}
	})
}
