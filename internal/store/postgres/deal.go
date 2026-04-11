package postgres

import (
	"context"
	"fmt"
	"time"

	"dealdet/internal/domain"
)

func (s *Store) InsertDealCandidate(ctx context.Context, candidate domain.DealCandidate) error {
	pool, err := s.poolOrError()
	if err != nil {
		return err
	}

	var expiresAt any
	if candidate.ExpiresAt != nil {
		expiresAt = *candidate.ExpiresAt
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO deal_candidate (
			id,
			normalized_listing_id,
			canonical_product_id,
			deal_score,
			pct_below_baseline,
			abs_saving_usd,
			baseline_price,
			status,
			detected_at,
			expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (normalized_listing_id) DO NOTHING
	`,
		candidate.ID,
		candidate.NormalizedListingID,
		candidate.CanonicalProductID,
		candidate.DealScore,
		candidate.PctBelowBaseline,
		candidate.AbsSavingUSD,
		candidate.BaselinePrice,
		string(candidate.Status),
		candidate.DetectedAt,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert deal candidate %s: %w", candidate.ID, err)
	}

	return nil
}

func (s *Store) ListDealCandidates(ctx context.Context, limit int) ([]domain.DealCandidate, error) {
	pool, err := s.poolOrError()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id,
			normalized_listing_id,
			canonical_product_id,
			deal_score,
			pct_below_baseline,
			abs_saving_usd,
			baseline_price,
			status,
			detected_at,
			expires_at
		FROM deal_candidate
		ORDER BY detected_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list deal candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.DealCandidate, 0, limit)
	for rows.Next() {
		var candidate domain.DealCandidate
		var status string
		var expiresAt *time.Time

		if err := rows.Scan(
			&candidate.ID,
			&candidate.NormalizedListingID,
			&candidate.CanonicalProductID,
			&candidate.DealScore,
			&candidate.PctBelowBaseline,
			&candidate.AbsSavingUSD,
			&candidate.BaselinePrice,
			&status,
			&candidate.DetectedAt,
			&expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan deal candidate: %w", err)
		}

		candidate.Status = domain.DealStatus(status)
		candidate.ExpiresAt = expiresAt
		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deal candidates: %w", err)
	}

	return candidates, nil
}