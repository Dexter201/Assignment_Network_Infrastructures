package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config defines the specific configuration for the Gateway
type Config struct {
	Port           string `yaml:"port"`
	CertPath       string `yaml:"cert_path"`
	KeyPath        string `yaml:"key_path"`
	UserServiceURL string `yaml:"user_service_url"`
	PostServiceURL string `yaml:"post_service_url"`
	FeedServiceURL string `yaml:"feed_service_url"`
}

// LoadConfig reads and parses a YAML file into the provided struct
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
