package normalization

import (
	"context"
	"testing"
	"time"

	"dealdet/internal/domain"
	"dealdet/internal/normalization/resolver"

	"github.com/google/uuid"
)

func TestPipelineProcessResolvesConditionAndEntity(t *testing.T) {
	productID := uuid.New()
	pipeline := NewPipeline([]resolver.MatchRule{
		{
			Name:               "sony-85",
			CanonicalProductID: productID,
			RequiredKeywords:   []string{"sony", "85mm"},
			MatchMode:          resolver.MatchModeAll,
			Priority:           10,
		},
	})

	raw := domain.RawListing{
		ID:              uuid.New(),
		SourceID:        uuid.New(),
		SourceListingID: "listing-1",
		Title:           "Sony FE 85mm f/1.8",
		ConditionRaw:    "Very Good with scratches",
		PriceCents:      12345,
		Currency:        "USD",
		ListingType:     domain.ListingTypeActive,
		URL:             "https://example.com/1",
		FetchedAt:       time.Now().UTC(),
	}

	normalized, err := pipeline.Process(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized.RawListingID != raw.ID {
		t.Fatalf("expected raw listing id %s, got %s", raw.ID, normalized.RawListingID)
	}
	if normalized.CanonicalProductID == nil {
		t.Fatalf("expected canonical product match")
	}
	if *normalized.CanonicalProductID != productID {
		t.Fatalf("expected canonical product id %s, got %s", productID, *normalized.CanonicalProductID)
	}
	if normalized.ConditionCanonical != domain.ConditionGood {
		t.Fatalf("expected downgraded condition good, got %q", normalized.ConditionCanonical)
	}
	if normalized.PriceUSD != 123.45 {
		t.Fatalf("expected price usd 123.45, got %v", normalized.PriceUSD)
	}
}

func TestPipelineProcessMarksUnresolvedWhenNoRuleMatch(t *testing.T) {
	pipeline := NewPipeline([]resolver.MatchRule{
		{
			Name:               "sony-85",
			CanonicalProductID: uuid.New(),
			RequiredKeywords:   []string{"sony", "85mm"},
			MatchMode:          resolver.MatchModeAll,
			Priority:           10,
		},
	})

	raw := domain.RawListing{
		ID:              uuid.New(),
		SourceID:        uuid.New(),
		SourceListingID: "listing-2",
		Title:           "Nikon Z 24-70",
		ConditionRaw:    "Used",
		PriceCents:      9999,
		Currency:        "USD",
		ListingType:     domain.ListingTypeSold,
		URL:             "https://example.com/2",
		FetchedAt:       time.Now().UTC(),
	}

	normalized, err := pipeline.Process(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized.CanonicalProductID != nil {
		t.Fatalf("expected unresolved listing with nil canonical product id")
	}
	if normalized.EntityConfidence != 0 {
		t.Fatalf("expected zero entity confidence for unresolved listing, got %v", normalized.EntityConfidence)
	}
}

func TestPipelineProcessRejectsMissingRawListingID(t *testing.T) {
	pipeline := NewPipeline(nil)

	_, err := pipeline.Process(context.Background(), domain.RawListing{})
	if err == nil {
		t.Fatalf("expected error for missing raw listing id")
	}
}
