package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Config defines the specific configuration for the Gateway, read from environment variables.
type Config struct {
	Port           string
	CertPath       string
	KeyPath        string
	UserServiceURL string
	PostServiceURL string
	FeedServiceURL string
	AuthDSN        string
	JWTSecret      string
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

	if cfg.Port == "" {
		cfg.Port = "8443"
		log.Printf("Defaulting to port %s", cfg.Port)
	}

	// Validate variables
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

// call the next middleware to do it's job
func callNextHandler(next http.Handler, writer http.ResponseWriter, receiver *http.Request) {
	next.ServeHTTP(writer, receiver)
}

// connect/ continously connect until healthcheck done to the db
func connectToDB(config *Config) (*sql.DB, error) {

	connStr := config.AuthDSN

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	//Ping the database to verify the connection
	//and close connection if error -->log
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to the database.")
	return db, nil
}

// test function
func healthcheck(mux *http.ServeMux) {
	//healthz is a standard way to name health check endpoints
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, receiver *http.Request) {
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})
}

// create the Table
func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}
	log.Println("Database table 'users' verified successfully.")
	return nil
}
