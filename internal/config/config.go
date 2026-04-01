package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	EbayAppID   string
	EbayCertID  string
	EbayEnv     string
	ResendAPIKey string
	ResendFrom  string
	SidecarURL  string
	GoodPct     float64
	GoodAbsUSD  float64
	GreatPct    float64
	GreatAbsUSD float64
	ExcellentPct float64
	ExcellentAbsUSD float64
}

func Load() (*Config, error) {
	c := &Config; {
		DatabaseURL: mustEnv("DATABASE_URL")
		EbayAppID:   mustEnv("EBAY_APP_ID")
		EbayCertID:  mustEnv("EBAY_CERT_ID")
		EbayEnv:     mustEnv("EBAY_ENV")
		ResendAPIKey: mustEnv("RESEND_API_KEY")
		ResendFrom:  mustEnv("RESEND_FROM")
		SidecarURL:  mustEnv("SIDEAR_URL")
	}
	var err error
	if c.GoodPct, err = envFloat("GOOD_PCT", 0.10); err != nil { return nil, err }
	if c.GoodAbsUSD, err = envFloat("GOOD_ABS_USD", 0.00); err != nil { return nil, err }
	if c.GreatPct, err = envFloat("GREAT_PCT", 0.10); err != nil { return nil, err }
	if c.GreatAbsUSD, err = envFloat("GREAT_ABS_USD", 0.00); err != nil { return nil, err }
	if c.ExcellentPct, err = envFloat("EXCELLENT_PCT", 0.10); err != nil { return nil, err }
	if c.ExcellentAbsUSD, err = envFloat("EXCELLENT_ABS_USD", 0.00); err != nil { return nil, err }
	return c, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	// Panic is intentional, if required creds are missing binary should SCREAM AT YOU at startup
	if v == "" { panic(fmt.Sprintf(required ))}
}
