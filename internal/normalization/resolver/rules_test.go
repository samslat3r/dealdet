package resolver

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeTitle(t *testing.T) {
	normalized := NormalizeTitle("Sony FE 85mm f/1.8 (Excellent!)")
	want := "sony fe 85mm f 1 8 excellent"
	if normalized != want {
		t.Fatalf("expected %q, got %q", want, normalized)
	}
}

func TestMatchTitleUsesPriorityAndRequiredKeywords(t *testing.T) {
	sonyID := uuid.New()
	canonID := uuid.New()

	rules := []MatchRule{
		{
			Name:               "canon",
			CanonicalProductID: canonID,
			RequiredKeywords:   []string{"canon", "50mm"},
			ExcludedKeywords:   []string{"fd"},
			MatchMode:          MatchModeAll,
			Priority:           5,
		},
		{
			Name:               "sony",
			CanonicalProductID: sonyID,
			RequiredKeywords:   []string{"sony", "85mm"},
			MatchMode:          MatchModeAll,
			Priority:           10,
		},
	}

	id, confidence := MatchTitle("Sony FE 85mm f/1.8", rules)
	if id == nil {
		t.Fatalf("expected product match")
	}
	if *id != sonyID {
		t.Fatalf("expected sony product id, got %s", *id)
	}
	if confidence <= 0 {
		t.Fatalf("expected positive confidence, got %v", confidence)
	}
}

func TestMatchTitleRespectsExcludedKeywords(t *testing.T) {
	canonID := uuid.New()
	rules := []MatchRule{
		{
			Name:               "canon-rf-50",
			CanonicalProductID: canonID,
			RequiredKeywords:   []string{"canon", "50mm"},
			ExcludedKeywords:   []string{"fd"},
			MatchMode:          MatchModeAll,
			Priority:           10,
		},
	}

	id, confidence := MatchTitle("Canon FD 50mm f/1.8", rules)
	if id != nil {
		t.Fatalf("expected no match due to excluded keyword, got %s", *id)
	}
	if confidence != 0 {
		t.Fatalf("expected zero confidence for no match, got %v", confidence)
	}
}

func TestMatchTitleNoRulesMatch(t *testing.T) {
	rules := []MatchRule{
		{
			Name:               "sony-85",
			CanonicalProductID: uuid.New(),
			RequiredKeywords:   []string{"sony", "85mm"},
			MatchMode:          MatchModeAll,
			Priority:           1,
		},
	}

	id, confidence := MatchTitle("Nikon Z 24-70", rules)
	if id != nil {
		t.Fatalf("expected no match, got %s", *id)
	}
	if confidence != 0 {
		t.Fatalf("expected zero confidence, got %v", confidence)
	}
}
