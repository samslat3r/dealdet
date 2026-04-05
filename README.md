# dealdet - deal detective

Deal detection used (in MVP, for camera lenses). Monitors (in MVP) eBay for underpriced listings.
It scores them against a baseline from sold prices. It will alert you by email Discord, or SMS.

---

## Stack

- **Go Worker** - polls ebay every 2m (active) and 6h (for sold listings), normalizes listings, scores deals, sends alerts
- **Go API** - REST on :8090
- **Python sidecar** - local ML for entity resolution and product condition mapping
- **PostgreSQL** - everything
- **SvelteKit** *(planned - see bottom - not part of MVP)* user registration, watch lists, deal feeds

---

## MVP Quick Start (when its done)


```bash
# 1. Prerequisites
#    Go 1.22+, PostgreSQL 15/16, uv (astral.sh/uv), golang-migrate

# 2. Configure
cp .env.example .env          # fill in eBay, Resend, DB creds

# 3. Database
docker-compose up -d postgres
make migrate-up

# 4. Sidecar
make sidecar-sync             # uv installs deps + downloads models
make sidecar-run              # FastAPI on :8080

# 5. Worker + API (separate terminals)
make run-worker
make run-api

# 6. Verify
curl http://localhost:8090/health
curl http://localhost:8090/deals | jq .
```


---

## Environment variables

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `EBAY_APP_ID` | yes | eBay developer App ID |
| `EBAY_CERT_ID` | yes | eBay developer Cert ID |
| `EBAY_ENVIRONMENT` | no | `production` (default) or `sandbox` |
| `RESEND_API_KEY` | no | Email alerts via Resend |
| `RESEND_FROM` | no | Sending address |
| `DISCORD_WEBHOOK_URL` | no | Discord alerts (great + excellent tier) |
| `TWILIO_ACCOUNT_SID` | no | SMS alerts (excellent tier only) |
| `TWILIO_AUTH_TOKEN` | no | |
| `TWILIO_FROM_NUMBER` | no | |
| `SIDECAR_URL` | no | defaults to `http://localhost:8080` |

## Alert tiers

Both thresholds must be met for an alert to be triggered This is just MVP stuff so its customizable.

| Tier | % below baseline | AND saving | Channels |
|---|---|---|---|
| Good | ≥ 10% | ≥ $15 | Email |
| Great | ≥ 20% | ≥ $40 | Email + Discord |
| Excellent | ≥ 30% | ≥ $75 | Email + Discord + SMS |

Defaults live in `.env`. Override per product in the database:

```sql
UPDATE watch_target SET good_pct = 0.12, good_abs_usd = 20 WHERE id = '...';
```

---

## Adding a product

(Again, this is just MVP. In the future, users will be able to add products and customize thresholds via the UI.)
"Slug" is part of the URL path. on eBay. Example: https://www.ebay.com/itm/<slug>/<itemId>

```sql
INSERT INTO canonical_product (slug, brand, model, mount, category)
VALUES ('sony-fe-85-f18', 'Sony', 'FE 85mm f/1.8', 'sony-e', 'lens');

INSERT INTO watch_target (watchlist_id, canonical_product_id)
SELECT w.id, cp.id FROM watchlist w, canonical_product cp
WHERE cp.slug = 'sony-fe-85-f18' LIMIT 1;
```

```bash
curl -X POST http://localhost:8080/reload   # refresh sidecar embeddings
```

## TODO for README:

- Adding a marketplace
- Diagnosing the pipeline
- Make targets
- SvelteKit frontend plan, endpoints, middleware, structure, etc.
- Page descriptions (i.e. `/` `/register`)
- Setup & What to build first
