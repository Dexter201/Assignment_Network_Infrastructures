package main

import (
	"log"
	"net/http"
)

// Config struct for feed-service

func main() {
	// 1. Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Set up a simple handler (you'll implement this later)
	// This just logs that a request was received for now
	http.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Feed service: Received request for /feed")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Feed service is running"))
	})

	// 3. Start the blocking HTTP server
	log.Printf("Feed service listening on :%s (HTTP)", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Feed service server failed: %v", err)
	}
}
