package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port      string `yaml:"port"`
	UserLBURL string `yaml:"user_lb_url"`
	PostLBURL string `yaml:"post_lb_url"`
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
