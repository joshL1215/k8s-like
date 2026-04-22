package httpx

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteErr(w http.ResponseWriter, code int, msg string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	WriteJSON(w, code, map[string]string{"error": msg, "detail": detail})
}
