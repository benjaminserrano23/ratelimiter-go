package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Store  StoreConfig  `yaml:"store"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type StoreConfig struct {
	Type     string `yaml:"type"` // "memory" or "redis"
	RedisURL string `yaml:"redis_url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), nil
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Store.Type == "" {
		cfg.Store.Type = "memory"
	}

	// Environment variable overrides (for Docker/production)
	if t := os.Getenv("STORE_TYPE"); t != "" {
		cfg.Store.Type = t
	}
	if u := os.Getenv("REDIS_URL"); u != "" {
		cfg.Store.RedisURL = u
	}
	if p := os.Getenv("PORT"); p != "" {
		cfg.Server.Port = p
	}

	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: "8080"},
		Store:  StoreConfig{Type: "memory"},
	}
}
