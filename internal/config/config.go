package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL     string
	EbayAppID       string
	EbayCertID      string
	EbayEnv         string
	ResendAPIKey    string
	ResendFrom      string
	SidecarURL      string
	GoodPct         float64
	GoodAbsUSD      float64
	GreatPct        float64
	GreatAbsUSD     float64
	ExcellentPct    float64
	ExcellentAbsUSD float64
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:  mustEnv("DATABASE_URL"),
		EbayAppID:    mustEnv("EBAY_APP_ID"),
		EbayCertID:   mustEnv("EBAY_CERT_ID"),
		EbayEnv:      firstNonEmpty("production", "EBAY_ENVIRONMENT", "EBAY_ENV"),
		ResendAPIKey: firstNonEmpty("", "RESEND_API_KEY"),
		ResendFrom:   firstNonEmpty("", "RESEND_FROM"),
		SidecarURL:   firstNonEmpty("http://localhost:8080", "SIDECAR_URL"),
	}

	if c.EbayEnv != "production" && c.EbayEnv != "sandbox" {
		return nil, fmt.Errorf("EBAY_ENVIRONMENT must be \"production\" or \"sandbox\", got %q", c.EbayEnv)
	}

	var err error
	if c.GoodPct, err = envFloat("GOOD_PCT", 0.10); err != nil {
		return nil, err
	}
	if c.GoodAbsUSD, err = envFloat("GOOD_ABS_USD", 15.00); err != nil {
		return nil, err
	}
	if c.GreatPct, err = envFloat("GREAT_PCT", 0.20); err != nil {
		return nil, err
	}
	if c.GreatAbsUSD, err = envFloat("GREAT_ABS_USD", 40.00); err != nil {
		return nil, err
	}
	if c.ExcellentPct, err = envFloat("EXCELLENT_PCT", 0.30); err != nil {
		return nil, err
	}
	if c.ExcellentAbsUSD, err = envFloat("EXCELLENT_ABS_USD", 75.00); err != nil {
		return nil, err
	}
	return c, nil
}

func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	// Panic is intentional, if required creds are missing binary should SCREAM AT YOU at startup
	if v == "" {
		panic(fmt.Sprintf("missing required environment variable %q", key))
	}
	return v
}

func firstNonEmpty(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return defaultValue
}

func envFloat(key string, defaultValue float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s=%q: %w", key, raw, err)
	}
	return value, nil
}
