// examples/metrics-example/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ruslano69/shp/middleware"
	"github.com/ruslano69/shp/pkg/converter"
)

func main() {
	// Создаем middleware с метриками
	config := middleware.Config{
		EnableCache:   true,
		EnableMetrics: true,
		Options: converter.Options{
			AutoFix: true,
		},
	}

	mux := http.NewServeMux()

	// HTML страница для демонстрации
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<HTML><BODY><H1>Test</H1><BR><P>Content</P></BODY></HTML>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Endpoint для метрик
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		stats := config.GetMetrics()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_conversions":      stats.TotalConversions,
			"successful_conversions": stats.SuccessfulConversions,
			"failed_conversions":     stats.FailedConversions,
			"average_duration_ms":    stats.AverageDuration.Milliseconds(),
			"total_bytes_processed":  stats.TotalBytesProcessed,
			"total_bytes_output":     stats.TotalBytesOutput,
			"changes_applied":        stats.ChangesApplied,
			"errors_by_type":         stats.ErrorsByType,
		})
	})

	// Применяем middleware
	handler := middleware.XHTMLMiddleware(config)(mux)

	fmt.Println("Server starting on :8080")
	fmt.Println("Visit http://localhost:8080/ for conversion")
	fmt.Println("Visit http://localhost:8080/metrics for stats")
	
	log.Fatal(http.ListenAndServe(":8080", handler))
}
