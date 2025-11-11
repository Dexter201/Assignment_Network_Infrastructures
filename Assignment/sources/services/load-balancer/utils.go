package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port      string
	Algorithm string
	Backends  []string
	Rate      float64
}

// LoadConfig reads and parses configuration from environment variables
func LoadConfig() (*Config, error) {

	port := os.Getenv("LB_PORT")
	algorithm := os.Getenv("LB_ALGORITHM")
	backendsStr := os.Getenv("LB_BACKENDS")
	rateStr := os.Getenv("LB_RATE")

	if port == "" {
		port = "8080" // default
		log.Printf("Defaulting to port %s", port)
	}

	if algorithm == "" {
		algorithm = "roundrobin" // Default to roundrobin
		log.Printf("Defaulting to algorithm %s", algorithm)
	}
	// Validate algorithm
	if algorithm != "roundrobin" && algorithm != "leastconn" && algorithm != "hashing" {
		return nil, errors.New("invalid algorithm: must be roundrobin, leastconn, or hashing")
	}

	if backendsStr == "" {
		return nil, errors.New("LB_BACKENDS environment variable is not set")
	}

	backends := strings.Split(backendsStr, ",")
	if len(backends) == 0 {
		return nil, errors.New("no backends provided")
	}
	log.Printf("Loaded backends: %v", backends)

	// --- Rate ---
	var rate float64 = 100.0 // Default 100 MB/s
	if rateStr != "" {
		var err error
		rate, err = strconv.ParseFloat(rateStr, 64)
		if err != nil {
			return nil, errors.New("invalid LB_RATE: must be a number")
		}
	}

	cfg := &Config{
		Port:      port,
		Algorithm: algorithm,
		Backends:  backends,
		Rate:      rate,
	}

	return cfg, nil
}
