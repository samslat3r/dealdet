package resolver

import (
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	MatchModeAll = "all"
	MatchModeAny = "any"
)

type MatchRule struct {
	Name               string
	CanonicalProductID uuid.UUID
	RequiredKeywords   []string
	ExcludedKeywords   []string
	MatchMode          string
	Priority           int
}

// NormalizeTitle lower-cases a title, strips punctuation, and collapses whitespace.
func NormalizeTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return ""
	}

	b := strings.Builder{}
	b.Grow(len(title))
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}

	return strings.Join(strings.Fields(b.String()), " ")
}

// MatchTitle deterministically resolves a canonical product from normalized keyword rules.
func MatchTitle(title string, rules []MatchRule) (*uuid.UUID, float64) {
	normalized := NormalizeTitle(title)
	if normalized == "" || len(rules) == 0 {
		return nil, 0
	}

	ordered := orderedRules(rules)
	for _, rule := range ordered {
		if !ruleMatches(normalized, rule) {
			continue
		}

		id := rule.CanonicalProductID
		return &id, confidenceForRule(rule)
	}

	return nil, 0
}

func orderedRules(rules []MatchRule) []MatchRule {
	ordered := make([]MatchRule, len(rules))
	copy(ordered, rules)

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority > ordered[j].Priority
		}
		if len(ordered[i].RequiredKeywords) != len(ordered[j].RequiredKeywords) {
			return len(ordered[i].RequiredKeywords) > len(ordered[j].RequiredKeywords)
		}
		if ordered[i].CanonicalProductID != ordered[j].CanonicalProductID {
			return ordered[i].CanonicalProductID.String() < ordered[j].CanonicalProductID.String()
		}
		return ordered[i].Name < ordered[j].Name
	})

	return ordered
}

func ruleMatches(normalizedTitle string, rule MatchRule) bool {
	if matchesAnyPhrase(normalizedTitle, rule.ExcludedKeywords) {
		return false
	}

	return matchesRequired(normalizedTitle, rule.RequiredKeywords, rule.MatchMode)
}

func matchesRequired(normalizedTitle string, required []string, mode string) bool {
	if len(required) == 0 {
		return false
	}

	if strings.EqualFold(mode, MatchModeAny) {
		return matchesAnyPhrase(normalizedTitle, required)
	}

	for _, keyword := range required {
		if !containsPhrase(normalizedTitle, keyword) {
			return false
		}
	}
	return true
}

func matchesAnyPhrase(normalizedTitle string, phrases []string) bool {
	for _, phrase := range phrases {
		if containsPhrase(normalizedTitle, phrase) {
			return true
		}
	}
	return false
}

func containsPhrase(normalizedTitle, phrase string) bool {
	normPhrase := NormalizeTitle(phrase)
	if normPhrase == "" {
		return false
	}
	return strings.Contains(" "+normalizedTitle+" ", " "+normPhrase+" ")
}

func confidenceForRule(rule MatchRule) float64 {
	base := 0.90
	if strings.EqualFold(rule.MatchMode, MatchModeAny) {
		base = 0.82
	}

	bonus := 0.02 * float64(minInt(len(rule.RequiredKeywords), 3))
	confidence := base + bonus
	if confidence > 0.99 {
		return 0.99
	}
	return confidence
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
