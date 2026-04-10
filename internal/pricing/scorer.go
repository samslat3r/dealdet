package pricing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

// Scorer compares active listings vs references gathered by baseline

// ScoreResult carries the deal candidate and metadata about how scoring resolved.
type ScoreResult struct {
	Candidate  domain.DealCandidate
	Suppressed bool   // true when the candidate was stored but should not trigger alerts
	Reason     string // human-readable suppression reason, empty when not suppressed
}

// ScoreCandidate converts a resolved active normalized listing into a DealCandidate.
// Unresolved products (nil CanonicalProductID) are suppressed with StatusInsufficient.
// Products with insufficient sold history are scored at zero and stored as StatusInsufficient.
func ScoreCandidate(ctx context.Context, lister SoldPriceLister, listing domain.NormalizedListing, windowDays int) (ScoreResult, error) {
	now := time.Now().UTC()

	// Unresolved product — store but suppress from alerting.
	if listing.CanonicalProductID == nil {
		candidate := domain.DealCandidate{
			ID:                  uuid.New(),
			NormalizedListingID: listing.ID,
			DealScore:           0,
			PctBelowBaseline:    0,
			AbsSavingUSD:        0,
			BaselinePrice:       0,
			Status:              domain.StatusInsufficient,
			DetectedAt:          now,
		}
		return ScoreResult{
			Candidate:  candidate,
			Suppressed: true,
			Reason:     "unresolved canonical product",
		}, nil
	}

	productID := *listing.CanonicalProductID

	baseline, err := ComputeBaseline(ctx, lister, productID, listing.ConditionCanonical, windowDays)
	if err != nil {
		if errors.Is(err, ErrInsufficientData) {
			candidate := domain.DealCandidate{
				ID:                  uuid.New(),
				NormalizedListingID: listing.ID,
				CanonicalProductID:  productID,
				DealScore:           0,
				PctBelowBaseline:    0,
				AbsSavingUSD:        0,
				BaselinePrice:       0,
				Status:              domain.StatusInsufficient,
				DetectedAt:          now,
			}
			return ScoreResult{
				Candidate:  candidate,
				Suppressed: true,
				Reason:     fmt.Sprintf("insufficient sold history for product %s", productID),
			}, nil
		}
		return ScoreResult{}, fmt.Errorf("score candidate %s: %w", listing.ID, err)
	}

	score, pctBelow, absSaving := domain.Score(listing.PriceUSD, baseline.Snapshot.TrimmedMeanPriceUSD)

	status := domain.StatusCandidate
	if score <= 0 {
		status = domain.StatusExpired // priced at or above baseline — not a deal
	}

	candidate := domain.DealCandidate{
		ID:                  uuid.New(),
		NormalizedListingID: listing.ID,
		CanonicalProductID:  productID,
		DealScore:           score,
		PctBelowBaseline:    pctBelow,
		AbsSavingUSD:        absSaving,
		BaselinePrice:       baseline.Snapshot.TrimmedMeanPriceUSD,
		Status:              status,
		DetectedAt:          now,
	}

	suppressed := status != domain.StatusCandidate
	reason := ""
	if suppressed {
		reason = "listing priced at or above baseline"
	}

	return ScoreResult{
		Candidate:  candidate,
		Suppressed: suppressed,
		Reason:     reason,
	}, nil
}
