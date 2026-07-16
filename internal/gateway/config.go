package gateway

import "os"

// Config holds runtime settings for the HTTP gateway.
type Config struct {
	ListenAddr  string
	DatabaseURL string
}

// ConfigFromEnv loads gateway configuration from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{
		ListenAddr:  getenv("GATEWAY_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable"),
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
