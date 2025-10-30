package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config struct for load-balancer
type Config struct {
	Port      string   `yaml:"port"`
	Algorithm string   `yaml:"algorithm"`
	Backends  []string `yaml:"backends"`
	Rate      float64  `yaml:"rate"`
}

// LoadConfig reads and parses the config.yaml file
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
