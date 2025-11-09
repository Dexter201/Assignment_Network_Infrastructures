package feed

import (
	"errors"
	"log"
	"net/http"
	"os"
)

// Config holds configuration for the feed-service
type Config struct {
	Port      string
	UserLBURL string
	PostLBURL string
}

// LoadConfig reads and parses configuration from environment variables
func LoadConfig() (*Config, error) {

	cfg := &Config{
		Port:      os.Getenv("FEED_SERVICE_PORT"),
		UserLBURL: os.Getenv("USER_SERVICE_URL"),
		PostLBURL: os.Getenv("POST_SERVICE_URL"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080" // A default if not set
		log.Printf("Feed service: Defaulting to port %s", cfg.Port)
	}

	if cfg.UserLBURL == "" {
		return nil, errors.New("USER_SERVICE_URL environment variable is not set")
	}
	if cfg.PostLBURL == "" {
		return nil, errors.New("POST_SERVICE_URL environment variable is not set")
	}

	log.Println("Feed service configuration loaded successfully")
	return cfg, nil
}

func healthcheck(mux *http.ServeMux) {
	//healthz is a standard way to name health check endpoints
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, receiver *http.Request) {
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})
}
