package domain

import (
	"time"

	"github.com/google/uuid"
)

// 5-tier condition scale
type ConditionTier string

const (
	ConditionNew        ConditionTier = "new"
	ConditionLikeNew    ConditionTier = "like_new"
	ConditionVeryGood   ConditionTier = "very_good"
	ConditionGood       ConditionTier = "good"
	ConditionAcceptable ConditionTier = "acceptable"
	ConditionUnknown    ConditionTier = "unknown"
)

// Rank returns numeric rank for comparison
func (c ConditionTier) Rank() int {
	switch c {
	case ConditionNew:
		return 5
	case ConditionLikeNew:
		return 4
	case ConditionVeryGood:
		return 3
	case ConditionGood:
		return 2
	case ConditionAcceptable:
		return 1
	default:
		return 0
	}
}

// Downgrade returns next lower condition tier
func (c ConditionTier) Downgrade() ConditionTier {
	switch c {
	case ConditionNew:
		return ConditionLikeNew
	case ConditionLikeNew:
		return ConditionVeryGood
	case ConditionVeryGood:
		return ConditionGood
	case ConditionGood:
		return ConditionAcceptable
	case ConditionAcceptable:
		return ConditionAcceptable
	default:
		return ConditionUnknown
	}
}

// for sale or sold
type ListingType string

const (
	ListingTypeActive ListingType = "active"
	ListingTypeSold   ListingType = "sold"
)

// RawListing is VERBATIM data from marketplace API reply . Contract between adapter and normalization pipeline
type RawListing struct {
	ID              uuid.UUID
	SourceID        uuid.UUID
	SourceListingID string
	Title           string
	ConditionRaw    string
	PriceCents      int64 // need this in cents USD normalized
	Currency        string
	ListingType     ListingType
	URL             string
	FetchedAt       time.Time
}

// NormalizedListing is output of pipeline of RawListing
// CanonicalProductID is nil when entity resolution fails
type NormalizedListing struct {
	ID                  uuid.UUID
	RawListingID        uuid.UUID
	CanonicalProductID  *uuid.UUID
	ConditionCanonical  ConditionTier
	ConditionMethod     string  // "ebay_enum" | keyword_downgrade | "classifier" | "Classifier fallback enum" or others
	ConditionConfidence float64 // 1.0 for rule based, model confidence for ML sidecar
	PriceUSD            float64
	EntityConfidence    float64 //cos similarity from embedding model ... is that accurate?
	/*
		 * https://arxiv.org/pdf/2403.05440
			* Cosine-similarity is the cosine of the angle between two vectors, or
			* equivalently the dot product between their normalizations.... we caution against blindly
			* using cosine-similarity and outline alternatives.
	*/
}

// Records computed baseline for product+condition pair
// Rows are append only; query MAX(computed_at) for the current baseline
type PriceSnapshot struct {
	ID                  uuid.UUID
	CanonicalProductID  uuid.UUID
	ConditionCanonical  ConditionTier
	TrimmedMeanPriceUSD float64
	SampleSize          int
	WindowDays          int
	ComputedAt          time.Time
}
