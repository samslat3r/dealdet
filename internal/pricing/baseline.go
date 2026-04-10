package pricing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

/*
Serves to compute a reference market price from sold listings, before active listings are scored.

	Active listings will be judged against this. The main job here is ComputeBaseline which wants SoldPriceLister for
	a given product and condition. Defaults to a 90 day window. < 8 samples falls back to querying all conditions for the product. Still < 8 samples results in an error.
	The baseline price is a trimmed mean of the sold prices, with 10% trim on each end (if sample size allows).

	Once enough sold prices are gathered, trimmedMean is used to compute the baseline price. Packages into domain.Pricesnapshot
*/
const (
	MinSampleSize     = 8
	DefaultWindowDays = 90
	TrimPct           = 0.10
)

// SoldPriceLister returns sold normalized-listing prices for a given product and condition
// within a lookback window. The store layer implements this interface.
type SoldPriceLister interface {
	ListSoldPrices(ctx context.Context, canonicalProductID uuid.UUID, condition domain.ConditionTier, windowDays int) ([]float64, error)
}

// BaselineResult holds the outcome of a baseline computation, including whether
// a condition fallback was used.
type BaselineResult struct {
	Snapshot domain.PriceSnapshot
	Fallback bool // true when exact condition had < MinSampleSize and a broader query was used
}

// ComputeBaseline computes a trimmed-mean baseline price for the given product
// and condition.
func ComputeBaseline(ctx context.Context, lister SoldPriceLister, productID uuid.UUID, cond domain.ConditionTier, windowDays int) (BaselineResult, error) {
	if windowDays <= 0 {
		windowDays = DefaultWindowDays
	}

	prices, err := lister.ListSoldPrices(ctx, productID, cond, windowDays)
	if err != nil {
		return BaselineResult{}, fmt.Errorf("baseline: list sold prices for %s/%s: %w", productID, cond, err)
	}

	fallback := false
	if len(prices) < MinSampleSize {
		// Condition fallback: query all conditions for this product.
		prices, err = lister.ListSoldPrices(ctx, productID, "", windowDays)
		if err != nil {
			return BaselineResult{}, fmt.Errorf("baseline: list sold prices fallback for %s: %w", productID, err)
		}
		fallback = true
	}

	if len(prices) < MinSampleSize {
		return BaselineResult{}, fmt.Errorf("baseline: product %s condition %s: %w (have %d, need %d)",
			productID, cond, ErrInsufficientData, len(prices), MinSampleSize)
	}

	mean := trimmedMean(prices, TrimPct)

	snapshot := domain.PriceSnapshot{
		ID:                  uuid.New(),
		CanonicalProductID:  productID,
		ConditionCanonical:  cond,
		TrimmedMeanPriceUSD: mean,
		SampleSize:          len(prices),
		WindowDays:          windowDays,
		ComputedAt:          time.Now().UTC(),
	}

	return BaselineResult{Snapshot: snapshot, Fallback: fallback}, nil
}

// Sold listing data is noisy - trim the top and bottom 10% of prices to mitigate outliers. If the sample is too small, skip trimming.
func trimmedMean(values []float64, trimPct float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	trimCount := int(math.Floor(float64(len(sorted)) * trimPct))
	if len(sorted)-2*trimCount < 3 {
		trimCount = 0
	}

	trimmed := sorted[trimCount : len(sorted)-trimCount]
	var sum float64
	for _, v := range trimmed {
		sum += v
	}
	return sum / float64(len(trimmed))
}

var ErrInsufficientData = fmt.Errorf("insufficient sold data")
