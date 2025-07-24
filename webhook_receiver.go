package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	defaultWebhookReadTimeout  = 5 * time.Second
	defaultWebhookWriteTimeout = 10 * time.Second
	defaultWebhookIdleTimeout  = 120 * time.Second
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		log.Println("Received webhook:")
		log.Printf("  User ID: %.0f", data["user_id"])
		log.Printf("  Old IP: %s", data["old_ip"])
		log.Printf("  New IP: %s", data["new_ip"])
		log.Printf("  User Agent: %s", data["user_agent"])

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Webhook received!")); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	})

	server := &http.Server{
		Addr:         ":9090",
		Handler:      mux,
		ReadTimeout:  defaultWebhookReadTimeout,
		WriteTimeout: defaultWebhookWriteTimeout,
		IdleTimeout:  defaultWebhookIdleTimeout,
	}

	log.Println("Webhook receiver listening on :9090")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
