package ebay

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"dealdet/internal/domain"
	"dealdet/internal/ingestion"
)

const (
	adapterName = "ebay"
)

var _ ingestion.MarketplaceSource = (*Adapter)(nil)

type Adapter struct {
	browse  *browseClient
	finding *findingClient
}

func init() {
	ingestion.Registry[adapterName] = New
}

func New(cfg ingestion.AdapterConfig) (ingestion.MarketplaceSource, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("ebay adapter: app id is required")
	}
	if cfg.CertID == "" {
		return nil, fmt.Errorf("ebay adapter: cert id is required")
	}

	env := cfg.Env
	if env == "" {
		env = "production"
	}
	if env != "production" && env != "sandbox" {
		return nil, fmt.Errorf("ebay adapter: invalid env %q", env)
	}

	httpClient := &http.Client{Timeout: 25 * time.Second}
	tokenCache := &tokenCache{}

	return &Adapter{
		browse: &browseClient{
			baseURL:    browseBaseURL(env),
			tokens:     tokenCache,
			appID:      cfg.AppID,
			certID:     cfg.CertID,
			env:        env,
			sourceID:   cfg.SourceID,
			httpClient: httpClient,
		},
		finding: &findingClient{
			baseURL:    findingBaseURL(env),
			appID:      cfg.AppID,
			env:        env,
			sourceID:   cfg.SourceID,
			httpClient: httpClient,
		},
	}, nil
}

func (a *Adapter) Name() string {
	return adapterName
}

func (a *Adapter) Fetch(ctx context.Context, params ingestion.FetchParams) ([]domain.RawListing, error) {
	if a == nil || a.browse == nil || a.finding == nil {
		return nil, fmt.Errorf("ebay adapter is not initialized")
	}

	switch params.ListingType {
	case domain.ListingTypeActive:
		return a.browse.FetchActive(ctx, params.Query, params.MaxItems)
	case domain.ListingTypeSold:
		return a.finding.FetchSold(ctx, params.Query, params.MaxItems)
	default:
		return nil, fmt.Errorf("ebay adapter: unsupported listing type %q", params.ListingType)
	}
}

func (a *Adapter) RateLimit(ctx context.Context) (remaining int, resetAt time.Time, err error) {
	_ = ctx
	// eBay rate limit introspection is endpoint-specific; return unknown until scheduler policy is wired.
	return -1, time.Time{}, nil
}

func browseBaseURL(env string) string {
	if env == "sandbox" {
		return "https://api.sandbox.ebay.com"
	}
	return "https://api.ebay.com"
}

func findingBaseURL(env string) string {
	if env == "sandbox" {
		return "https://svcs.sandbox.ebay.com/services/search/FindingService"
	}
	return "https://svcs.ebay.com/services/search/FindingService"
}
