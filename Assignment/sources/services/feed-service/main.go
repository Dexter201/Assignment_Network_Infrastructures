package main

import (
	"log"
	"net/http"
)

func createRouter(handler *FeedHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/feed", handler)

	// Passive health check endpoint for Docker
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, receiver *http.Request) {
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})

	return mux
}

func main() {

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	handler := createFeedHandler(config)

	mux := createRouter(handler)

	// 4. Start the HTTP server
	log.Printf("Feed service listening on :%s (HTTP)", config.Port)
	if err := http.ListenAndServe(":"+config.Port, mux); err != nil {
		log.Fatalf("Feed service server failed: %v", err)
	}
}
