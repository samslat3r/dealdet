package condition

import (
	"strings"
	"unicode"

	"dealdet/internal/domain"
)

type Resolution struct {
	Tier       domain.ConditionTier
	Method     string
	Confidence float64
}

var ebayConditionMap = map[string]domain.ConditionTier{
	"new":                      domain.ConditionNew,
	"new other":                domain.ConditionLikeNew,
	"new other see details":    domain.ConditionLikeNew,
	"open box":                 domain.ConditionLikeNew,
	"like new":                 domain.ConditionLikeNew,
	"very good":                domain.ConditionVeryGood,
	"good":                     domain.ConditionGood,
	"acceptable":               domain.ConditionAcceptable,
	"used":                     domain.ConditionGood,
	"pre owned":                domain.ConditionGood,
	"preowned":                 domain.ConditionGood,
	"seller refurbished":       domain.ConditionVeryGood,
	"manufacturer refurbished": domain.ConditionVeryGood,
	"certified refurbished":    domain.ConditionVeryGood,
	"for parts or not working": domain.ConditionAcceptable,
}

var severeConditionKeywords = []string{
	"for parts",
	"not working",
	"broken",
	"faulty",
	"as is",
	"as-is",
	"parts only",
}

var downgradeConditionKeywords = []string{
	"damage",
	"damaged",
	"scratch",
	"scratches",
	"scuff",
	"haze",
	"fungus",
	"mold",
	"dent",
	"dented",
	"repair",
	"blemish",
	"wear",
}

// Resolve maps raw eBay condition text to a canonical tier and applies conservative keyword downgrades.
func Resolve(rawCondition string) Resolution {
	normalized := normalize(rawCondition)
	if normalized == "" {
		return Resolution{Tier: domain.ConditionUnknown, Method: "unresolved_condition", Confidence: 0}
	}

	if containsAny(normalized, severeConditionKeywords) {
		return Resolution{Tier: domain.ConditionAcceptable, Method: "keyword_override", Confidence: 0.95}
	}

	tier, method, confidence := resolveBaseTier(normalized)
	if containsAny(normalized, downgradeConditionKeywords) && tier != domain.ConditionAcceptable && tier != domain.ConditionUnknown {
		return Resolution{Tier: tier.Downgrade(), Method: "keyword_downgrade", Confidence: minFloat(confidence, 0.90)}
	}

	return Resolution{Tier: tier, Method: method, Confidence: confidence}
}

func resolveBaseTier(normalized string) (domain.ConditionTier, string, float64) {
	if tier, ok := ebayConditionMap[normalized]; ok {
		return tier, "ebay_enum", 1.0
	}

	if containsAny(normalized, []string{"open box", "like new"}) {
		return domain.ConditionLikeNew, "keyword_map", 0.85
	}
	if containsAny(normalized, []string{"very good", "excellent"}) {
		return domain.ConditionVeryGood, "keyword_map", 0.85
	}
	if containsAny(normalized, []string{"good", "used", "pre owned", "preowned"}) {
		return domain.ConditionGood, "keyword_map", 0.80
	}
	if containsAny(normalized, []string{"new"}) {
		return domain.ConditionNew, "keyword_map", 0.80
	}

	return domain.ConditionUnknown, "unresolved_condition", 0.30
}

func containsAny(normalized string, terms []string) bool {
	for _, term := range terms {
		if containsPhrase(normalized, term) {
			return true
		}
	}
	return false
}

func containsPhrase(normalized, phrase string) bool {
	normPhrase := normalize(phrase)
	if normPhrase == "" {
		return false
	}
	return strings.Contains(" "+normalized+" ", " "+normPhrase+" ")
}

func normalize(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}

	b := strings.Builder{}
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}

	return strings.Join(strings.Fields(b.String()), " ")
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
