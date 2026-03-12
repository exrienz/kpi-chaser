package config

import "os"

type Config struct {
	HTTPAddress       string
	DatabasePath      string
	JWTSecret         string
	LLMAPIKey         string
	LLMBaseURL        string
	LLMModel          string
	WorkerConcurrency int
}

func Load() Config {
	return Config{
		HTTPAddress:       getEnv("HTTP_ADDRESS", ":8080"),
		DatabasePath:      getEnv("DATABASE_PATH", "./kpi-journal.db"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me"),
		LLMAPIKey:         os.Getenv("LLM_API_KEY"),
		LLMBaseURL:        getEnv("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMModel:          getEnv("LLM_MODEL", "gpt-4o-mini"),
		WorkerConcurrency: 3,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
