package postgres

import (
	"context"
	"fmt"
	"time"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

type MatchRuleRow struct {
	ID                 uuid.UUID
	CanonicalProductID uuid.UUID
	Name               string
	RequiredKeywords   []string
	ExcludedKeywords   []string
	MatchMode          string
	Priority           int
	Active             bool
	CreatedAt          time.Time
}

func (s *Store) InsertRawListing(ctx context.Context, listing domain.RawListing) error {
	pool, err := s.poolOrError()
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO raw_listing (
			id,
			source_id,
			source_listing_id,
			title,
			condition_raw,
			price_cents,
			currency,
			listing_type,
			url,
			fetched_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (source_id, source_listing_id, listing_type) DO NOTHING
	`,
		listing.ID,
		listing.SourceID,
		listing.SourceListingID,
		listing.Title,
		listing.ConditionRaw,
		listing.PriceCents,
		listing.Currency,
		string(listing.ListingType),
		listing.URL,
		listing.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert raw listing %s: %w", listing.SourceListingID, err)
	}

	return nil
}

func (s *Store) InsertNormalizedListing(ctx context.Context, listing domain.NormalizedListing) error {
	pool, err := s.poolOrError()
	if err != nil {
		return err
	}

	var canonicalProductID any
	if listing.CanonicalProductID != nil {
		canonicalProductID = *listing.CanonicalProductID
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO normalized_listing (
			id,
			raw_listing_id,
			canonical_product_id,
			condition_canonical,
			condition_method,
			condition_confidence,
			price_usd,
			entity_confidence
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (raw_listing_id) DO NOTHING
	`,
		listing.ID,
		listing.RawListingID,
		canonicalProductID,
		string(listing.ConditionCanonical),
		listing.ConditionMethod,
		listing.ConditionConfidence,
		listing.PriceUSD,
		listing.EntityConfidence,
	)
	if err != nil {
		return fmt.Errorf("insert normalized listing %s: %w", listing.ID, err)
	}

	return nil
}

func (s *Store) ListCanonicalProducts(ctx context.Context) ([]domain.CanonicalProduct, error) {
	pool, err := s.poolOrError()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, slug, brand, model, mount, category, created_at
		FROM canonical_product
		ORDER BY brand, model, slug
	`)
	if err != nil {
		return nil, fmt.Errorf("list canonical products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.CanonicalProduct, 0)
	for rows.Next() {
		var product domain.CanonicalProduct
		if err := rows.Scan(
			&product.ID,
			&product.Slug,
			&product.Brand,
			&product.Model,
			&product.Mount,
			&product.Category,
			&product.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan canonical product: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate canonical products: %w", err)
	}

	return products, nil
}

func (s *Store) ListMatchRules(ctx context.Context) ([]MatchRuleRow, error) {
	pool, err := s.poolOrError()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, canonical_product_id, name, required_keywords, excluded_keywords, match_mode, priority, active, created_at
		FROM match_rule
		WHERE active = TRUE
		ORDER BY canonical_product_id, priority DESC, created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list match rules: %w", err)
	}
	defer rows.Close()

	rules := make([]MatchRuleRow, 0)
	for rows.Next() {
		var rule MatchRuleRow
		if err := rows.Scan(
			&rule.ID,
			&rule.CanonicalProductID,
			&rule.Name,
			&rule.RequiredKeywords,
			&rule.ExcludedKeywords,
			&rule.MatchMode,
			&rule.Priority,
			&rule.Active,
			&rule.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan match rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate match rules: %w", err)
	}

	return rules, nil
}

func (s *Store) ListWatchTargetsByProduct(ctx context.Context, canonicalProductID uuid.UUID) ([]domain.WatchTargetRow, error) {
	pool, err := s.poolOrError()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT
			wt.id,
			wt.watchlist_id,
			wt.canonical_product_id,
			wt.good_pct,
			wt.good_abs_usd,
			wt.great_pct,
			wt.great_abs_usd,
			wt.excellent_pct,
			wt.excellent_abs_usd,
			wt.notify_in_app,
			wt.notify_email,
			wt.notify_discord,
			wt.notify_sms,
			wt.max_price_usd,
			wl.user_email,
			wl.user_phone,
			wl.discord_webhook_url
		FROM watch_target wt
		JOIN watchlist wl ON wl.id = wt.watchlist_id
		WHERE wt.canonical_product_id = $1
		ORDER BY wt.id
	`, canonicalProductID)
	if err != nil {
		return nil, fmt.Errorf("list watch targets for product %s: %w", canonicalProductID, err)
	}
	defer rows.Close()

	targets := make([]domain.WatchTargetRow, 0)
	for rows.Next() {
		var target domain.WatchTargetRow
		if err := rows.Scan(
			&target.ID,
			&target.WatchlistID,
			&target.CanonicalProductID,
			&target.GoodPct,
			&target.GoodAbsUSD,
			&target.GreatPct,
			&target.GreatAbsUSD,
			&target.ExcellentPct,
			&target.ExcellentAbsUSD,
			&target.NotifyInApp,
			&target.NotifyEmail,
			&target.NotifyDiscord,
			&target.NotifySMS,
			&target.MaxPriceUSD,
			&target.UserEmail,
			&target.UserPhone,
			&target.DiscordWebhookURL,
		); err != nil {
			return nil, fmt.Errorf("scan watch target: %w", err)
		}
		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watch targets: %w", err)
	}

	return targets, nil
}

func (s *Store) InsertPriceSnapshot(ctx context.Context, snapshot domain.PriceSnapshot) error {
	pool, err := s.poolOrError()
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO price_snapshot (
			id,
			canonical_product_id,
			condition_canonical,
			trimmed_mean_price_usd,
			sample_size,
			window_days,
			computed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		snapshot.ID,
		snapshot.CanonicalProductID,
		string(snapshot.ConditionCanonical),
		snapshot.TrimmedMeanPriceUSD,
		snapshot.SampleSize,
		snapshot.WindowDays,
		snapshot.ComputedAt,
	)
	if err != nil {
		return fmt.Errorf("insert price snapshot %s: %w", snapshot.ID, err)
	}

	return nil
}