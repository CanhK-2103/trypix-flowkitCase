package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL, Address, ArtifactDir, OpenRouterKey, FalKey string
	OpenRouterURL, FalURL, FalPricingURL                     string
	Concurrency                                              int
	PollInterval, RetryBase, LLMDeadline, ImageDeadline      time.Duration
}

func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if ok && os.Getenv(strings.TrimSpace(key)) == "" {
			_ = os.Setenv(strings.TrimSpace(key), strings.Trim(strings.TrimSpace(val), "\"'"))
		}
	}
	return s.Err()
}
func Env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func Load() Config {
	c, _ := strconv.Atoi(Env("WORKER_CONCURRENCY", "4"))
	if c < 1 {
		c = 4
	}
	return Config{DatabaseURL: Env("DATABASE_URL", "postgres://flowkit:flowkit@localhost:5432/flowkit?sslmode=disable"), Address: ":" + Env("PORT", "8080"), ArtifactDir: Env("ARTIFACT_DIR", "artifacts"), OpenRouterKey: os.Getenv("OPENROUTER_API_KEY"), FalKey: os.Getenv("FAL_KEY"), OpenRouterURL: Env("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"), FalURL: Env("FAL_QUEUE_BASE_URL", "https://queue.fal.run"), FalPricingURL: Env("FAL_PRICING_BASE_URL", "https://api.fal.ai/v1/models/pricing"), Concurrency: c, PollInterval: 2 * time.Second, RetryBase: 2 * time.Second, LLMDeadline: 120 * time.Second, ImageDeadline: 600 * time.Second}
}
