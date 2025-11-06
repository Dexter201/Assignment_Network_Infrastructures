package main

import (
	"errors"
	"fmt"
	"log"
	"os"
)

// Config defines the specific configuration for the Gateway, read from environment variables.
type Config struct {
	Port           string
	CertPath       string
	KeyPath        string
	UserServiceURL string
	PostServiceURL string
	FeedServiceURL string
	AuthDSN        string // For the Auth DB
	JWTSecret      string // For signing JWTs
}

// LoadConfig reads and parses configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:           os.Getenv("GATEWAY_PORT"),
		CertPath:       os.Getenv("GATEWAY_CERT_PATH"),
		KeyPath:        os.Getenv("GATEWAY_KEY_PATH"),
		UserServiceURL: os.Getenv("USER_SERVICE_URL"),
		PostServiceURL: os.Getenv("POST_SERVICE_URL"),
		FeedServiceURL: os.Getenv("FEED_SERVICE_URL"),
		AuthDSN:        os.Getenv("AUTH_POSTGRES_DSN"),
		JWTSecret:      os.Getenv("JWT_SECRET_KEY"),
	}

	// Set default port
	if cfg.Port == "" {
		cfg.Port = "8443"
		log.Printf("Defaulting to port %s", cfg.Port)
	}

	// Validate critical variables
	if cfg.CertPath == "" {
		return nil, errors.New("GATEWAY_CERT_PATH environment variable is not set")
	}
	if cfg.KeyPath == "" {
		return nil, errors.New("GATEWAY_KEY_PATH environment variable is not set")
	}
	if cfg.AuthDSN == "" {
		return nil, errors.New("AUTH_POSTGRES_DSN environment variable is not set")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET_KEY environment variable is not set")
	}
	if cfg.UserServiceURL == "" || cfg.PostServiceURL == "" || cfg.FeedServiceURL == "" {
		return nil, errors.New("one or more service URLs are not set")
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

// Helper function to get a required env var or fail
func getEnvOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}
