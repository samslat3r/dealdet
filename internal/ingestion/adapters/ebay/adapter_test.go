package ebay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dealdet/internal/domain"
	"dealdet/internal/ingestion"

	"github.com/google/uuid"
)

func TestRegistryIncludesEbayAdapter(t *testing.T) {
	constructor, ok := ingestion.Registry[adapterName]
	if !ok || constructor == nil {
		t.Fatalf("expected %q adapter constructor to be registered", adapterName)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       ingestion.AdapterConfig
		wantError bool
	}{
		{
			name:      "missing app id",
			cfg:       ingestion.AdapterConfig{CertID: "cert", Env: "production"},
			wantError: true,
		},
		{
			name:      "missing cert id",
			cfg:       ingestion.AdapterConfig{AppID: "app", Env: "production"},
			wantError: true,
		},
		{
			name:      "invalid env",
			cfg:       ingestion.AdapterConfig{AppID: "app", CertID: "cert", Env: "dev"},
			wantError: true,
		},
		{
			name:      "valid default env",
			cfg:       ingestion.AdapterConfig{AppID: "app", CertID: "cert", SourceID: uuid.New()},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := New(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if source == nil {
				t.Fatalf("expected source to be initialized")
			}
		})
	}
}

func TestFetchRejectsUnsupportedListingType(t *testing.T) {
	adapter := &Adapter{
		browse:  &browseClient{},
		finding: &findingClient{},
	}

	_, err := adapter.Fetch(context.Background(), ingestion.FetchParams{
		Query:       "lens",
		MaxItems:    1,
		ListingType: domain.ListingType("unknown"),
	})
	if err == nil {
		t.Fatalf("expected unsupported listing type error")
	}
}

func TestFetchActiveRoutesToBrowseClient(t *testing.T) {
	sourceID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"itemSummaries":[{"itemId":"123","title":"Sony FE 85mm","price":{"value":"123.45","currency":"USD"},"condition":"Used","itemWebUrl":"https://example.com/123"}],"total":1,"limit":1}`))
	}))
	defer server.Close()

	adapter := &Adapter{
		browse: &browseClient{
			baseURL:  server.URL,
			tokens:   &tokenCache{token: "token", expiresAt: time.Now().Add(time.Hour)},
			appID:    "app",
			certID:   "cert",
			env:      "production",
			sourceID: sourceID,
			httpClient: &http.Client{
				Timeout: 3 * time.Second,
			},
		},
		finding: &findingClient{},
	}

	listings, err := adapter.Fetch(context.Background(), ingestion.FetchParams{
		Query:       "sony lens",
		MaxItems:    1,
		ListingType: domain.ListingTypeActive,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	if listings[0].ListingType != domain.ListingTypeActive {
		t.Fatalf("expected active listing type, got %q", listings[0].ListingType)
	}
	if listings[0].SourceID != sourceID {
		t.Fatalf("expected source id %s, got %s", sourceID, listings[0].SourceID)
	}
}

func TestFetchSoldRoutesToFindingClient(t *testing.T) {
	sourceID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`
			<findCompletedItemsResponse>
				<ack>Success</ack>
				<searchResult count="1">
					<item>
						<itemId>abc-1</itemId>
						<title>Canon RF 50mm</title>
						<sellingStatus>
							<currentPrice currencyId="USD">99.99</currentPrice>
						</sellingStatus>
						<condition>
							<conditionDisplayName>Used</conditionDisplayName>
						</condition>
						<viewItemURL>https://example.com/abc-1</viewItemURL>
					</item>
				</searchResult>
				<paginationOutput>
					<totalEntries>1</totalEntries>
					<totalPages>1</totalPages>
					<entriesPerPage>1</entriesPerPage>
					<pageNumber>1</pageNumber>
				</paginationOutput>
			</findCompletedItemsResponse>
		`))
	}))
	defer server.Close()

	adapter := &Adapter{
		browse: &browseClient{},
		finding: &findingClient{
			baseURL:    server.URL,
			appID:      "app",
			env:        "production",
			sourceID:   sourceID,
			httpClient: &http.Client{Timeout: 3 * time.Second},
		},
	}

	listings, err := adapter.Fetch(context.Background(), ingestion.FetchParams{
		Query:       "canon lens",
		MaxItems:    1,
		ListingType: domain.ListingTypeSold,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	if listings[0].ListingType != domain.ListingTypeSold {
		t.Fatalf("expected sold listing type, got %q", listings[0].ListingType)
	}
	if listings[0].SourceID != sourceID {
		t.Fatalf("expected source id %s, got %s", sourceID, listings[0].SourceID)
	}
}

func TestRateLimitReturnsUnknownSentinel(t *testing.T) {
	adapter := &Adapter{}
	remaining, resetAt, err := adapter.RateLimit(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if remaining != -1 {
		t.Fatalf("expected unknown sentinel -1, got %d", remaining)
	}
	if !resetAt.IsZero() {
		t.Fatalf("expected zero reset time")
	}
}
