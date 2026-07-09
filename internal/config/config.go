package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken    string
	BotAdminIDs []int64

	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string

	MetricsHTTPPort string
	MetricsBaseURL  string

	PostgresDSN   string
	MigrationsDir string

	HTTPClientTimeout time.Duration
}

func Load() Config {
	cfg := Config{
		BotToken:          env("BOT_TOKEN", ""),
		BotAdminIDs:       parseAdminIDs(env("BOT_ADMIN_IDS", "")),
		LLMAPIKey:         env("LLM_API_KEY", ""),
		LLMBaseURL:        env("LLM_BASE_URL", ""),
		LLMModel:          env("LLM_MODEL", "skald-loki"),
		MetricsHTTPPort:   env("METRICS_HTTP_PORT", "8080"),
		MetricsBaseURL:    strings.TrimRight(env("METRICS_BASE_URL", "http://localhost:8080"), "/"),
		MigrationsDir:     env("MIGRATIONS_DIR", "migrations"),
		HTTPClientTimeout: 30 * time.Second,
	}
	cfg.PostgresDSN = postgresDSN()
	return cfg
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseAdminIDs(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func postgresDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		return dsn
	}

	user := env("POSTGRES_USER", "linkedout")
	password := env("POSTGRES_PASSWORD", "linkedout")
	db := env("POSTGRES_DB", "linkedout")
	host := env("POSTGRES_HOST", "localhost")
	port := env("POSTGRES_PORT", "5432")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, db)
}
