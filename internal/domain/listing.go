package domain
// STUB
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
	ConditionAcceptable ConditionTier = "acceptable"
	ConditionUnknown    ConditionTier = "unknown"
)

// Rank returns numeric rank for comparison
//
func (c ConditionTier) Rank() int {
	switch c {
	case ConditionNew:
		return 5
	case ConditionLikeNew:
		return 4
	case ConditionVeryGood:
		return 3
	case ConditionAcceptable:
		return 2
	case ConditionUnknown:
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
		return ConditionAcceptable
	case ConditionAcceptable:
	default:
		return ConditionAcceptable
	}
}


// for sale or sold
type ListingType string
const (
	ListingTypeActive  ListingType = "active"
	ListingTypeSold     ListingType = "sold"
)

// Rawlisting is VERBATIM data from marketplace API reply . Contract between adapter and normalization pipeline
type RawListing struct {

}
// NormalizedListing is output of pipeline of RawListing
// CanonicalProductID is nil when entity resolution fails
type NormalizedListing struct {
}

// Records computed baseline for product+condition pair
// Rows are append only; query MAX(computed_at) for the current baseline
type PriceSnapshot struct {
}
