package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

type browseClient struct {
	baseURL    string
	tokens     *tokenCache
	appID      string
	certID     string
	env        string
	sourceID   uuid.UUID
	httpClient *http.Client
}

// ebay Browse API response shapes.
type browseItem struct {
	ItemID string `json:"itemId"`
	Title  string `json:"title"`
	Price  struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"price"`
	Condition  string `json:"condition"`
	ItemWebURL string `json:"itemWebUrl"`
}

type browseResponse struct {
	ItemSummaries []browseItem `json:"itemSummaries"`
	Total         int          `json:"total"`
	Limit         int          `json:"limit"`
}

// fetchPage performs one Browse API search request.
func (c *browseClient) fetchPage(ctx context.Context, query string, offset, limit int) (*browseResponse, error) {
	token, err := c.tokens.get(ctx, c.appID, c.certID, c.env)
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"q":            {query},
		"limit":        {strconv.Itoa(limit)},
		"offset":       {strconv.Itoa(offset)},
		"filter":       {"buyingOptions:{FIXED_PRICE},deliveryCountry:US"},
		"category_ids": {"625"},
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/buy/browse/v1/item_summary/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("ebay browse: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ebay browse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebay browse: status %d", resp.StatusCode)
	}

	var out browseResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ebay browse: decode response: %w", err)
	}

	return &out, nil
}

// FetchActive fetches active fixed-price listings up to maxItems.
func (c *browseClient) FetchActive(ctx context.Context, query string, maxItems int) ([]domain.RawListing, error) {
	if maxItems <= 0 {
		return []domain.RawListing{}, nil
	}

	const pageSize = 200
	listings := make([]domain.RawListing, 0, maxItems)

	for offset := 0; len(listings) < maxItems; offset += pageSize {
		remaining := maxItems - len(listings)
		limit := pageSize
		if remaining < pageSize {
			limit = remaining
		}

		page, err := c.fetchPage(ctx, query, offset, limit)
		if err != nil {
			return nil, err
		}
		if len(page.ItemSummaries) == 0 {
			break
		}

		now := time.Now().UTC()
		for _, item := range page.ItemSummaries {
			if len(listings) >= maxItems {
				break
			}

			priceCents, err := parsePriceCents(item.Price.Value)
			if err != nil {
				continue
			}

			listings = append(listings, domain.RawListing{
				ID:              uuid.New(),
				SourceID:        c.sourceID,
				SourceListingID: item.ItemID,
				Title:           item.Title,
				ConditionRaw:    item.Condition,
				PriceCents:      priceCents,
				Currency:        item.Price.Currency,
				ListingType:     domain.ListingTypeActive,
				URL:             item.ItemWebURL,
				FetchedAt:       now,
			})
		}

		if len(page.ItemSummaries) < limit {
			break
		}
	}

	return listings, nil
}

// parsePriceCents converts a decimal string like "123.45" to integer cents.
func parsePriceCents(raw string) (int64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse price %q: %w", raw, err)
	}
	return int64(v*100 + 0.5), nil
}
