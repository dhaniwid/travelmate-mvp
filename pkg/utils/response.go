package utils

import (
	"encoding/json"
	"net/http"
)

// WriteJSONResponse menulis data ke response writer dengan format JSON dan status code tertentu.
func WriteJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	// 1. Set Header Content-Type agar klien tahu ini adalah JSON
	w.Header().Set("Content-Type", "application/json")

	// 2. Set Status Code (misal: 200, 201, 400, 500)
	w.WriteHeader(status)

	// 3. Encode data ke JSON stream
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback jika terjadi error saat encoding (jarang terjadi)
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// WriteJSONError adalah wrapper khusus untuk mengirim pesan error standar.
// Format output: {"success": false, "error": "Pesan Error"}
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	payload := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	WriteJSONResponse(w, status, payload)
}
