package main

import (
	"log"
	"net/http"
)

// create a router for the feed handler to forward traffic to the correct endpoint
func createRouter(handler *FeedHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/feed", handler)

	// Passive health check endpoint for Docker --> debugging
	healthcheck(mux)

	return mux
}

// entrypoint for the feed service
func main() {

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	handler := createFeedHandler(config)

	mux := createRouter(handler)

	// Start the HTTP server
	log.Printf("Feed service listening on :%s (HTTP)", config.Port)
	if err := http.ListenAndServe(":"+config.Port, mux); err != nil {
		log.Fatalf("Feed service server failed: %v", err)
	}
}
