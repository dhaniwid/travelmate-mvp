// Ad-hoc tool: export existing trip to PDF (bypasses PRO auth — evaluation only).
// Usage: go run cmd/tools/export_pdf/main.go <tripID>
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"travelmate/internal/repositories"
	"travelmate/internal/services"
)

func main() {
	tripID := "f058520c-0642-4ce5-ace5-d65a294805c6"
	if len(os.Args) > 1 {
		tripID = os.Args[1]
	}

	dsn := "host=localhost port=5432 user=postgres dbname=travelmate sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	repo := repositories.NewTripRepository(db)
	pdfSvc := services.NewPDFService()

	ctx := context.Background()
	tripAndPlan, err := repo.GetTripWithPlan(ctx, tripID)
	if err != nil {
		log.Fatalf("GetTripWithPlan: %v", err)
	}
	if tripAndPlan == nil {
		log.Fatalf("trip not found: %s", tripID)
	}

	fmt.Printf("Trip: %s → %s (%d days)\n", tripAndPlan.Trip.ID, tripAndPlan.Trip.Destination, tripAndPlan.Trip.TripDays)
	fmt.Printf("Days in itinerary: %d\n", len(tripAndPlan.Plan.Itinerary))

	pdfBytes, err := pdfSvc.GenerateTripPDF(tripAndPlan.Trip, tripAndPlan.Plan)
	if err != nil {
		log.Fatalf("GenerateTripPDF: %v", err)
	}

	outPath := fmt.Sprintf("/tmp/Miru_Itinerary_%s.pdf", tripAndPlan.Trip.Destination)
	if err := os.WriteFile(outPath, pdfBytes, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}

	fmt.Printf("✅ PDF saved: %s (%d KB)\n", outPath, len(pdfBytes)/1024)
}
