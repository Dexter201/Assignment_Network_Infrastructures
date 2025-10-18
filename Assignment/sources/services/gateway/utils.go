package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port           string `yaml:"port"`
	JWTSecret      string `yaml:"jwt_secret"`
	DatabaseURL    string `yaml:"database_url"`
	UserServiceURL string `yaml:"user_service_url"`
	PostServiceURL string `yaml:"post_service_url"`
	FeedServiceURL string `yaml:"feed_service_url"`
	CertPath       string `yaml:"cert_path"`
	KeyPath        string `yaml:"key_path"`
}

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
