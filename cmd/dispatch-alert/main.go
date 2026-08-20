package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	dispatchalerts "example.com/fieldservice-dispatch-alerts"
)

func main() {
	client := &dispatchalerts.SMSClient{APIKey: os.Getenv("INFRAI_API_KEY"), MaxRetries: 3}
	handler := dispatchHandler{dispatcher: dispatchalerts.Dispatcher{SMS: client}}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /work-orders/dispatch", handler.serveHTTP)
	addr := envOr("ADDR", ":8080")
	log.Printf("dispatch alert service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

type dispatchHandler struct {
	dispatcher dispatchalerts.Dispatcher
}

func (h dispatchHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var input dispatchalerts.DispatchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return
	}

	result, err := h.dispatcher.Dispatch(r.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		var apiErr *dispatchalerts.APIError
		if errors.As(err, &apiErr) {
			status = apiErr.HTTPStatus
			if status < 400 || status >= 500 {
				status = http.StatusBadGateway
			}
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
