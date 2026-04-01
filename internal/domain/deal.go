package domain
// STUB

import (
	"math"
	"time"
	"github.com/google/uuid"
)

/// Quality tier of a deal candidate
type DealTier string

const (
	TierGood DealTier = "good"
	TierGreat DealTier = "great"
	TierExcellent DealTier = "excellent"
)


// Lifecycle of deal candidate
type DealStatus string
const (
	StatusCandidate DealStatus = "candidate"
	//stub

)

type DealCandidate struct {

}

type TierThresholds struct {

}

// Highestqualifyingtier returns best tier candidae qualifies for , returns ("" if it isn't even good)
func d *DealCandidate) HighestQualifyingTier(t TierThresholds) (DealTier, bool) {

}

// SCore computes normalized deal score & related metrics
// score is clamped to [0, 1]; 40% below baseline = 1.0
// returns zeros for invalid inputs (zero or negative baseline)
func Score(priceUSD, baselineUSD float64) (score, pctBelow, absSaving float64) {
	return
}
