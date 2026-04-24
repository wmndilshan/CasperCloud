package httpapi

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeData writes {"data": <payload>} for successful responses.
func writeData(w http.ResponseWriter, status int, payload any) {
	writeJSON(w, status, map[string]any{"data": payload})
}

// writeError writes {"error":{"message":"..."}}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}

// writeValidationError writes {"error":{"message":"...","fields":{field:msg}}}.
func writeValidationError(w http.ResponseWriter, message string, fields map[string]string) {
	errObj := map[string]any{"message": message}
	if len(fields) > 0 {
		errObj["fields"] = fields
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": errObj})
}
