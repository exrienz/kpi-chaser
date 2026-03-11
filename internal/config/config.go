package config

import "os"

type Config struct {
	HTTPAddress       string
	DatabasePath      string
	JWTSecret         string
	OpenRouterAPIKey  string
	OpenRouterBaseURL string
	OpenRouterModel   string
	WorkerConcurrency int
}

func Load() Config {
	return Config{
		HTTPAddress:       getEnv("HTTP_ADDRESS", ":8080"),
		DatabasePath:      getEnv("DATABASE_PATH", "./kpi-journal.db"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me"),
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterBaseURL: getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		OpenRouterModel:   getEnv("OPENROUTER_MODEL", "openai/gpt-4o-mini"),
		WorkerConcurrency: 3,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
