package main

import (
	"log"
	"net/http"
)

func main() {

	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Start HTTPS server
	log.Printf("Gateway listening on :%s (HTTPS)", config.Port)
	// Use the paths from the config struct, not hard-coded strings
	if err := http.ListenAndServeTLS(":"+config.Port, config.CertPath, config.KeyPath, nil); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}
