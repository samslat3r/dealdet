package pricing

import (
	"context"
	"testing"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

func TestScoreCandidate_UnresolvedProduct(t *testing.T) {
	lister := &stubLister{}
	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: nil,
		ConditionCanonical: domain.ConditionGood,
		PriceUSD:           150,
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Suppressed {
		t.Error("expected suppressed=true for unresolved product")
	}
	if result.Candidate.Status != domain.StatusInsufficient {
		t.Errorf("status = %s, want insufficient", result.Candidate.Status)
	}
	if result.Candidate.DealScore != 0 {
		t.Errorf("score = %f, want 0", result.Candidate.DealScore)
	}
}

func TestScoreCandidate_InsufficientHistory(t *testing.T) {
	pid := newProductID()
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": {100, 200},
			pid.String() + "|":     {100, 200},
		},
	}

	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: &pid,
		ConditionCanonical: domain.ConditionGood,
		PriceUSD:           50,
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Suppressed {
		t.Error("expected suppressed=true for insufficient history")
	}
	if result.Candidate.Status != domain.StatusInsufficient {
		t.Errorf("status = %s, want insufficient", result.Candidate.Status)
	}
	if result.Candidate.CanonicalProductID != pid {
		t.Error("product ID should still be set")
	}
}

func TestScoreCandidate_BelowBaseline(t *testing.T) {
	pid := newProductID()
	// 10 prices all at 200 → baseline ~200
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = 200
	}
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": prices,
		},
	}

	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: &pid,
		ConditionCanonical: domain.ConditionGood,
		PriceUSD:           150, // 25% below 200
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suppressed {
		t.Error("expected suppressed=false for good deal")
	}
	if result.Candidate.Status != domain.StatusCandidate {
		t.Errorf("status = %s, want candidate", result.Candidate.Status)
	}
	if result.Candidate.DealScore <= 0 {
		t.Error("score should be positive")
	}
	if result.Candidate.PctBelowBaseline <= 0 {
		t.Error("pct below baseline should be positive")
	}
	if result.Candidate.AbsSavingUSD <= 0 {
		t.Error("abs saving should be positive")
	}
	if result.Candidate.BaselinePrice != 200 {
		t.Errorf("baseline = %f, want 200", result.Candidate.BaselinePrice)
	}
}

func TestScoreCandidate_AtBaseline(t *testing.T) {
	pid := newProductID()
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = 200
	}
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": prices,
		},
	}

	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: &pid,
		ConditionCanonical: domain.ConditionGood,
		PriceUSD:           200, // at baseline — not a deal
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Suppressed {
		t.Error("expected suppressed=true for at-baseline price")
	}
	if result.Candidate.Status != domain.StatusExpired {
		t.Errorf("status = %s, want expired", result.Candidate.Status)
	}
}

func TestScoreCandidate_AboveBaseline(t *testing.T) {
	pid := newProductID()
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = 200
	}
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": prices,
		},
	}

	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: &pid,
		ConditionCanonical: domain.ConditionGood,
		PriceUSD:           250, // above baseline
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Suppressed {
		t.Error("expected suppressed=true for above-baseline price")
	}
	if result.Candidate.DealScore != 0 {
		t.Errorf("score = %f, want 0 for above-baseline", result.Candidate.DealScore)
	}
}

func TestScoreCandidate_40PctBelow_MaxScore(t *testing.T) {
	pid := newProductID()
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = 1000
	}
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|new": prices,
		},
	}

	listing := domain.NormalizedListing{
		ID:                 uuid.New(),
		RawListingID:       uuid.New(),
		CanonicalProductID: &pid,
		ConditionCanonical: domain.ConditionNew,
		PriceUSD:           600, // exactly 40% below 1000
	}

	result, err := ScoreCandidate(context.Background(), lister, listing, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Candidate.DealScore != 1.0 {
		t.Errorf("score = %f, want 1.0 for 40%% below baseline", result.Candidate.DealScore)
	}
}
