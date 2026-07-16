package gateway

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	sessionCookieName = "vigil_session"
	sessionTTL        = 7 * 24 * time.Hour
	maxUploadBytes    = 20 << 20
)

// Config holds runtime settings for the HTTP gateway.
type Config struct {
	ListenAddr          string
	DatabaseURL         string
	CORSOrigins         []string
	MaxConcurrentAudits int
	StageDelay          time.Duration // SSE fake stage delay (short in tests)
}

// ConfigFromEnv loads gateway configuration from environment variables.
func ConfigFromEnv() Config {
	delayMS, _ := strconv.Atoi(getenv("SSE_STAGE_DELAY_MS", "400"))
	maxConc, _ := strconv.Atoi(getenv("MAX_CONCURRENT_AUDITS", "4"))
	if maxConc < 1 {
		maxConc = 4
	}
	origins := strings.Split(getenv("CORS_ORIGINS", "http://localhost:3000"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return Config{
		ListenAddr:          getenv("GATEWAY_ADDR", ":8080"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable"),
		CORSOrigins:         origins,
		MaxConcurrentAudits: maxConc,
		StageDelay:          time.Duration(delayMS) * time.Millisecond,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
