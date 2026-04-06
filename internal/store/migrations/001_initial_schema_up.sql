CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE marketplace_source (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL UNIQUE,
	adapter_type TEXT NOT NULL,
	base_url TEXT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	rate_limit_daily INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE canonical_product (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	slug TEXT NOT NULL UNIQUE,
	brand TEXT NOT NULL,
	model TEXT NOT NULL,
	mount TEXT NOT NULL,
	category TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE match_rule (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	canonical_product_id UUID NOT NULL REFERENCES canonical_product(id) ON DELETE CASCADE,
	name TEXT NOT NULL DEFAULT '',
	required_keywords TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
	excluded_keywords TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
	match_mode TEXT NOT NULL DEFAULT 'all' CHECK (match_mode IN ('all', 'any')),
	priority INTEGER NOT NULL DEFAULT 0,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE watchlist (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL DEFAULT 'default',
	user_email TEXT NOT NULL DEFAULT '',
	user_phone TEXT NOT NULL DEFAULT '',
	discord_webhook_url TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE watch_target (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	watchlist_id UUID NOT NULL REFERENCES watchlist(id) ON DELETE CASCADE,
	canonical_product_id UUID NOT NULL REFERENCES canonical_product(id) ON DELETE CASCADE,
	good_pct DOUBLE PRECISION NOT NULL DEFAULT 0.10,
	good_abs_usd DOUBLE PRECISION NOT NULL DEFAULT 15.00,
	great_pct DOUBLE PRECISION NOT NULL DEFAULT 0.20,
	great_abs_usd DOUBLE PRECISION NOT NULL DEFAULT 40.00,
	excellent_pct DOUBLE PRECISION NOT NULL DEFAULT 0.30,
	excellent_abs_usd DOUBLE PRECISION NOT NULL DEFAULT 75.00,
	notify_in_app BOOLEAN NOT NULL DEFAULT FALSE,
	notify_email BOOLEAN NOT NULL DEFAULT TRUE,
	notify_discord BOOLEAN NOT NULL DEFAULT FALSE,
	notify_sms BOOLEAN NOT NULL DEFAULT FALSE,
	max_price_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (watchlist_id, canonical_product_id)
);

CREATE TABLE raw_listing (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	source_id UUID NOT NULL REFERENCES marketplace_source(id) ON DELETE RESTRICT,
	source_listing_id TEXT NOT NULL,
	title TEXT NOT NULL,
	condition_raw TEXT NOT NULL,
	price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
	currency TEXT NOT NULL,
	listing_type TEXT NOT NULL CHECK (listing_type IN ('active', 'sold')),
	url TEXT NOT NULL,
	fetched_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (source_id, source_listing_id, listing_type)
);

CREATE TABLE normalized_listing (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	raw_listing_id UUID NOT NULL UNIQUE REFERENCES raw_listing(id) ON DELETE CASCADE,
	canonical_product_id UUID REFERENCES canonical_product(id) ON DELETE SET NULL,
	condition_canonical TEXT NOT NULL CHECK (condition_canonical IN ('new', 'like_new', 'very_good', 'good', 'acceptable', 'unknown')),
	condition_method TEXT NOT NULL,
	condition_confidence DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (condition_confidence >= 0 AND condition_confidence <= 1),
	price_usd DOUBLE PRECISION NOT NULL CHECK (price_usd >= 0),
	entity_confidence DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (entity_confidence >= 0 AND entity_confidence <= 1),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE price_snapshot (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	canonical_product_id UUID NOT NULL REFERENCES canonical_product(id) ON DELETE CASCADE,
	condition_canonical TEXT NOT NULL CHECK (condition_canonical IN ('new', 'like_new', 'very_good', 'good', 'acceptable', 'unknown')),
	trimmed_mean_price_usd DOUBLE PRECISION NOT NULL CHECK (trimmed_mean_price_usd >= 0),
	sample_size INTEGER NOT NULL CHECK (sample_size >= 0),
	window_days INTEGER NOT NULL CHECK (window_days > 0),
	computed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE deal_candidate (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	normalized_listing_id UUID NOT NULL UNIQUE REFERENCES normalized_listing(id) ON DELETE CASCADE,
	canonical_product_id UUID NOT NULL REFERENCES canonical_product(id) ON DELETE CASCADE,
	deal_score DOUBLE PRECISION NOT NULL CHECK (deal_score >= 0 AND deal_score <= 1),
	pct_below_baseline DOUBLE PRECISION NOT NULL,
	abs_saving_usd DOUBLE PRECISION NOT NULL,
	baseline_price DOUBLE PRECISION NOT NULL CHECK (baseline_price >= 0),
	status TEXT NOT NULL CHECK (status IN ('candidate', 'alerted', 'expired', 'insufficient')),
	detected_at TIMESTAMPTZ NOT NULL,
	expires_at TIMESTAMPTZ
);

CREATE TABLE alert (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	watch_target_id UUID NOT NULL REFERENCES watch_target(id) ON DELETE CASCADE,
	deal_candidate_id UUID NOT NULL REFERENCES deal_candidate(id) ON DELETE CASCADE,
	tier TEXT NOT NULL CHECK (tier IN ('good', 'great', 'excellent')),
	channel TEXT NOT NULL CHECK (channel IN ('email', 'discord', 'sms', 'in_app')),
	idempotency_key TEXT NOT NULL UNIQUE,
	user_email TEXT NOT NULL DEFAULT '',
	user_phone TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
	error_message TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	sent_at TIMESTAMPTZ
);

CREATE INDEX idx_match_rule_canonical_product_id ON match_rule (canonical_product_id);
CREATE INDEX idx_watch_target_canonical_product_id ON watch_target (canonical_product_id);
CREATE INDEX idx_normalized_listing_canonical_product_id ON normalized_listing (canonical_product_id);
CREATE INDEX idx_price_snapshot_lookup ON price_snapshot (canonical_product_id, condition_canonical, computed_at DESC);
CREATE INDEX idx_deal_candidate_detected_at ON deal_candidate (detected_at DESC);
CREATE INDEX idx_alert_watch_target_id ON alert (watch_target_id);
CREATE INDEX idx_alert_deal_candidate_id ON alert (deal_candidate_id);

INSERT INTO marketplace_source (id, name, adapter_type, base_url, active, rate_limit_daily)
VALUES (
	'11111111-1111-1111-1111-111111111111',
	'ebay',
	'ebay',
	'https://api.ebay.com',
	TRUE,
	5000
)
ON CONFLICT (name) DO NOTHING;
