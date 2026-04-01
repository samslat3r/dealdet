package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// / Quality tier of a deal candidate
type DealTier string

const (
	TierGood      DealTier = "good"
	TierGreat     DealTier = "great"
	TierExcellent DealTier = "excellent"
)

// Lifecycle of deal candidate
type DealStatus string

const (
	StatusCandidate    DealStatus = "candidate"
	StatusAlerted      DealStatus = "alerted"
	StatusExpired      DealStatus = "expired"
	StatusInsufficient DealStatus = "insufficient"
)

type DealCandidate struct {
	ID                  uuid.UUID
	NormalizedListingID uuid.UUID
	CanonicalProductID  uuid.UUID
	DealScore           float64 // 0.0 - 1.0; 1.0 = 40% below baseline
	PctBelowBaseline    float64 // e.g. 0.25 = 25% below baseline
	AbsSavingUSD        float64
	BaselinePrice       float64
	Status              DealStatus
	DetectedAt          time.Time
	ExpiresAt           *time.Time // value is optional / present but zero potentially
}

type TierThresholds struct {
	GoodPct         float64
	GoodAbsUSD      float64
	GreatPct        float64
	GreatAbsUSD     float64
	ExcellentPct    float64
	ExcellentAbsUSD float64
}

// Highestqualifyingtier returns best tier candidate qualifies for , returns ("" if it isn't even good)
func (d *DealCandidate) HighestQualifyingTier(t TierThresholds) (DealTier, bool) {
	switch {
	case d.PctBelowBaseline >= t.ExcellentPct && d.AbsSavingUSD >= t.ExcellentAbsUSD:
		return TierExcellent, true
	case d.PctBelowBaseline >= t.GreatPct && d.AbsSavingUSD >= t.GreatAbsUSD:
		return TierGreat, true
	case d.PctBelowBaseline >= t.GoodPct && d.AbsSavingUSD >= t.GoodAbsUSD:
		return TierGood, true
	default:
		return "", false
	}
}

// Score computes normalized deal score & related metrics
// score is clamped to [0, 1]; 40% below baseline = 1.0
// returns zeros for invalid inputs (zero or negative baseline)
func Score(priceUSD, baselineUSD float64) (score, pctBelow, absSaving float64) {
	if baselineUSD <= 0 {
		return 0, 0, 0
	}
	pctBelow = (baselineUSD - priceUSD) / baselineUSD
	absSaving = baselineUSD - priceUSD
	score = math.Min(pctBelow/0.40, 1.0)
	if score < 0 {
		score = 0
	}
	return
}
