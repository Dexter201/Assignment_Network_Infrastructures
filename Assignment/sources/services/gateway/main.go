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
	if err := http.ListenAndServeTLS(":"+config.Port, "cert.pem", "key.pem", nil); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}
