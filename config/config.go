package config

import (
	"encoding/json"
	"log"
	"os"
)

// Config represents the application configuration
type Config struct {
	Database struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Name     string `json:"name"`
	} `json:"database"`
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetConfig returns the config, loading from file or environment variables
func GetConfig() *Config {
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Println("Config file not found, using environment variables")
		config = &Config{}
		config.Database.User = getEnv("DB_USER", "pashapay")
		config.Database.Password = getEnv("DB_PASSWORD", "Q1w2e3r4!@#")
		config.Database.Host = getEnv("DB_HOST", "192.168.46.4")
		config.Database.Port = getEnv("DB_PORT", "3306")
		config.Database.Name = getEnv("DB_NAME", "report")
		config.Server.Port = getEnv("SERVER_PORT", "5000")
	}

	return config
}

// Helper to get environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}