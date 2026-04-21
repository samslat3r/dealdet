package ingestion

import (
	"context"
	"dealdet/internal/domain"
	"time"

	"github.com/google/uuid"
)

// FetchParams

type FetchParams struct {
	Query       string
	MaxItems    int
	ListingType domain.ListingType
}

// MarketplaceSource is the contract each marketplace adapter needs to satisfy
// Scheduler only ever calls THIS interface - it never knows what marketplace directly
// They all need these methods anyway, so this is a good abstraction boundary
type MarketplaceSource interface {
	Name() string
	Fetch(ctx context.Context, params FetchParams) ([]domain.RawListing, error)
	RateLimit(ctx context.Context) (remaining int, resetAt time.Time, err error)
}

// AdapterConfig passed to adapter constructors`
// AdapterConfig holds the necessary configuration for an adapter
// SourceID is the UUID from the marketplace_source row set by the scheduler
type AdapterConfig struct {
	AppID    string
	CertID   string
	Env      string // prod? sandbox? dev?
	SourceID uuid.UUID
}

// Registry will map... adapter type strings to constructor functions ,
// Adapters register via init() in their own package, adding a new market works like this:
/*
	New Adapter Package
	New Registry Entry
	??? Nothing else that's it
*/

var Registry = map[string]func(cfg AdapterConfig) (MarketplaceSource, error){}
