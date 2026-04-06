package normalization

import (
	"context"
	"fmt"

	"dealdet/internal/domain"
	"dealdet/internal/normalization/condition"
	"dealdet/internal/normalization/resolver"

	"github.com/google/uuid"
)

type Pipeline struct {
	rules []resolver.MatchRule
}

func NewPipeline(rules []resolver.MatchRule) *Pipeline {
	rulesCopy := make([]resolver.MatchRule, len(rules))
	copy(rulesCopy, rules)

	return &Pipeline{rules: rulesCopy}
}

// Process runs deterministic condition resolution first, then deterministic entity resolution.
// Listings that do not match a rule are explicitly left unresolved (CanonicalProductID nil).
func (p *Pipeline) Process(ctx context.Context, raw domain.RawListing) (domain.NormalizedListing, error) {
	_ = ctx

	if raw.ID == uuid.Nil {
		return domain.NormalizedListing{}, fmt.Errorf("normalize listing: raw listing id is required")
	}

	conditionResult := condition.Resolve(raw.ConditionRaw)
	canonicalProductID, entityConfidence := resolver.MatchTitle(raw.Title, p.rules)

	return domain.NormalizedListing{
		ID:                  uuid.New(),
		RawListingID:        raw.ID,
		CanonicalProductID:  canonicalProductID,
		ConditionCanonical:  conditionResult.Tier,
		ConditionMethod:     conditionResult.Method,
		ConditionConfidence: conditionResult.Confidence,
		PriceUSD:            centsToUSD(raw.PriceCents),
		EntityConfidence:    entityConfidence,
	}, nil
}

func (p *Pipeline) ProcessBatch(ctx context.Context, rawListings []domain.RawListing) ([]domain.NormalizedListing, error) {
	normalized := make([]domain.NormalizedListing, 0, len(rawListings))
	for _, raw := range rawListings {
		listing, err := p.Process(ctx, raw)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, listing)
	}
	return normalized, nil
}

func centsToUSD(cents int64) float64 {
	return float64(cents) / 100.0
}
