package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
	"travelmate/internal/domain"
)

type PerformanceRepository struct {
	DB *sql.DB
}

func NewPerformanceRepository(db *sql.DB) *PerformanceRepository {
	return &PerformanceRepository{DB: db}
}

func (r *PerformanceRepository) SaveMetric(ctx context.Context, task string, duration time.Duration, dest, model string) {
	query := `
		INSERT INTO performance_metrics (task_name, duration_ms, destination, model_used) 
		VALUES ($1, $2, $3, $4)`

	_, err := r.DB.ExecContext(ctx, query, task, duration.Milliseconds(), dest, model)
	if err != nil {
		log.Printf("⚠️ [PerfRepo] Gagal simpan metriks: %v", err)
	}
}

// GetRecentStats mengambil agregasi data performa 24 jam terakhir
func (r *PerformanceRepository) GetRecentStats(ctx context.Context) ([]domain.PerformanceStats, error) {
	query := `
		SELECT 
			task_name, 
			ROUND(AVG(duration_ms), 2) as avg_ms,
			MAX(duration_ms) as max_ms,
			COUNT(*) as total_calls
		FROM performance_metrics 
		WHERE created_at > NOW() - INTERVAL '24 hours'
		GROUP BY task_name
		ORDER BY avg_ms DESC`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.PerformanceStats
	for rows.Next() {
		var s domain.PerformanceStats
		if err := rows.Scan(&s.TaskName, &s.AvgLatency, &s.MaxLatency, &s.TotalCalls); err != nil {
			return nil, err
		}
		stats = append(stats)
	}
	return stats, nil
}

// PrintStartupDashboard menampilkan tabel performa cantik di terminal
func (r *PerformanceRepository) PrintStartupDashboard() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := r.GetRecentStats(ctx)
	if err != nil {
		log.Printf("⚠️ Gagal memuat dashboard performa: %v", err)
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 62))
	fmt.Printf("🚀 TRAVELMATE PERFORMANCE DASHBOARD (Last 24h)\n")
	fmt.Println(strings.Repeat("-", 62))
	fmt.Printf("| %-15s | %-12s | %-12s | %-8s |\n", "Task Name", "Avg Latency", "Max Latency", "Calls")
	fmt.Println(strings.Repeat("-", 62))

	if len(stats) == 0 {
		fmt.Printf("| %-58s |\n", "No data recorded yet. Start planning some trips!")
	}

	for _, s := range stats {
		avgStr := fmt.Sprintf("%.2fs", s.AvgLatency/1000)
		maxStr := fmt.Sprintf("%.2fs", float64(s.MaxLatency)/1000)
		fmt.Printf("| %-15s | %-12s | %-12s | %-8d |\n",
			s.TaskName, avgStr, maxStr, s.TotalCalls)
	}

	fmt.Println(strings.Repeat("=", 62) + "\n")
}
