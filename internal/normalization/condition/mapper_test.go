package condition

import (
	"testing"

	"dealdet/internal/domain"
)

func TestResolveMapsKnownEbayCondition(t *testing.T) {
	result := Resolve("Very Good")
	if result.Tier != domain.ConditionVeryGood {
		t.Fatalf("expected very_good, got %q", result.Tier)
	}
	if result.Method != "ebay_enum" {
		t.Fatalf("expected ebay_enum method, got %q", result.Method)
	}
	if result.Confidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %v", result.Confidence)
	}
}

func TestResolveAppliesKeywordDowngrade(t *testing.T) {
	result := Resolve("Very Good, scratches on barrel")
	if result.Tier != domain.ConditionGood {
		t.Fatalf("expected downgraded tier good, got %q", result.Tier)
	}
	if result.Method != "keyword_downgrade" {
		t.Fatalf("expected keyword_downgrade method, got %q", result.Method)
	}
}

func TestResolveOverridesSevereCondition(t *testing.T) {
	result := Resolve("Used - for parts, not working")
	if result.Tier != domain.ConditionAcceptable {
		t.Fatalf("expected acceptable, got %q", result.Tier)
	}
	if result.Method != "keyword_override" {
		t.Fatalf("expected keyword_override method, got %q", result.Method)
	}
}

func TestResolveUnknownCondition(t *testing.T) {
	result := Resolve("pristine-ish maybe")
	if result.Tier != domain.ConditionUnknown {
		t.Fatalf("expected unknown, got %q", result.Tier)
	}
	if result.Method != "unresolved_condition" {
		t.Fatalf("expected unresolved_condition method, got %q", result.Method)
	}
}
