package domain

import (
	"time"

	"github.com/google/uuid"
)

// One distinct product --- This is designed for cameras in the MVP just because I want it to be hence the fields
type CanonicalProduct struct {
	ID        uuid.UUID
	Slug      string
	Brand     string
	Model     string
	Mount     string
	Category  string
	CreatedAt time.Time
}

// database representation of *a* marketplace source
type MarketplaceSourceRow struct {
	ID             uuid.UUID
	Name           string
	AdapterType    string
	BaseURL        string
	Active         bool
	RateLimitDaily int
}

// db representation of a single target. jointed to owning user's contact info for alerts
type WatchTargetRow struct {
	ID                 uuid.UUID
	WatchlistID        uuid.UUID
	CanonicalProductID uuid.UUID
	GoodPct            float64
	GoodAbsUSD         float64
	GreatPct           float64
	GreatAbsUSD        float64
	ExcellentPct       float64
	ExcellentAbsUSD    float64
	NotifyInApp        bool
	NotifyEmail        bool
	NotifyDiscord      bool
	NotifySMS          bool
	MaxPriceUSD        float64
	// fields for notifications - definitely change to .env loaded from config
	UserEmail         string
	UserPhone         string
	DiscordWebhookURL string
}

// Passed to the store layer for idempotent alerting
type AlertRow struct {
	WatchTargetID   uuid.UUID
	DealCandidateID uuid.UUID
	Tier            string
	Channel         string
	IdempotencyKey  string
	UserEmail       string
	UserPhone       string
}
