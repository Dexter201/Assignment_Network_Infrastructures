package main

import (
	"log"
	"net/http"
)

func main() {

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to the database
	db, err := connectToDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close() //defer sets db close in a waititng list and executes when the main ends aka the server stops running

	authHandler := createAuthHandler(db, []byte(config.JWTSecret))
	metricsHandler := createMetricsHandler()

	router, err := createRouter(authHandler, metricsHandler, config)
	if err != nil {
		log.Fatalf("Failed to create router: %w", err)
	}

	// Start the HTTPS server --> uses my self signed certificates
	log.Printf("Gateway listening on :%s (HTTPS)", config.Port)
	if err := http.ListenAndServeTLS(":"+config.Port, config.CertPath, config.KeyPath, router); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}
