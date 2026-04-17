package config 

import "os"

type Config struct {
	Port          string
	PostgresURL   string
	RedisURL      string
	AnthropicKey  string
}

func Load() *Config {
	return &Config {
		Port:         getEnv("PORT", "8080"),
		PostgresURL:  getEnv("POSTGRES_URL", ""),
		RedisURL:     getEnv("REDIS_URL", ""),
		AnthropicKey: getEnv("ANTHROPIC_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}