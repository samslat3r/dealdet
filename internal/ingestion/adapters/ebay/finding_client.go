package ebay

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

// eBay Finding API XML response shapes for findCompletedItems.
type findingResponse struct {
	XMLName      xml.Name            `xml:"findCompletedItemsResponse"`
	Ack          string              `xml:"ack"`
	SearchResult findingSearchResult `xml:"searchResult"`
	Pagination   findingPagination   `xml:"paginationOutput"`
}

type findingSearchResult struct {
	Count int           `xml:"count,attr"`
	Items []findingItem `xml:"item"`
}

type findingItem struct {
	ItemID        string               `xml:"itemId"`
	Title         string               `xml:"title"`
	SellingStatus findingSellingStatus `xml:"sellingStatus"`
	Condition     findingCondition     `xml:"condition"`
	ViewItemURL   string               `xml:"viewItemURL"`
}

type findingSellingStatus struct {
	CurrentPrice findingPrice `xml:"currentPrice"`
}

type findingPrice struct {
	Value      string `xml:",chardata"`
	CurrencyID string `xml:"currencyId,attr"`
}

type findingCondition struct {
	DisplayName string `xml:"conditionDisplayName"`
}

type findingPagination struct {
	TotalEntries   int `xml:"totalEntries"`
	TotalPages     int `xml:"totalPages"`
	EntriesPerPage int `xml:"entriesPerPage"`
	PageNumber     int `xml:"pageNumber"`
}

type findingClient struct {
	baseURL    string
	appID      string
	env        string
	sourceID   uuid.UUID
	httpClient *http.Client
}

// fetchPage calls findCompletedItems for one page (1-based page number).
func (c *findingClient) fetchPage(ctx context.Context, query string, page, limit int) (*findingResponse, error) {
	params := url.Values{
		"OPERATION-NAME":                 {"findCompletedItems"},
		"SERVICE-NAME":                   {"FindingService"},
		"SERVICE-VERSION":                {"1.0.0"},
		"SECURITY-APPNAME":               {c.appID},
		"RESPONSE-DATA-FORMAT":           {"XML"},
		"keywords":                       {query},
		"itemFilter(0).name":             {"SoldItemsOnly"},
		"itemFilter(0).value":            {"true"},
		"itemFilter(1).name":             {"ListingType"},
		"itemFilter(1).value":            {"FixedPrice"},
		"paginationInput.entriesPerPage": {strconv.Itoa(limit)},
		"paginationInput.pageNumber":     {strconv.Itoa(page)},
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("ebay finding: build request: %w", err)
	}
	req.Header.Set("X-EBAY-SOA-SECURITY-APPNAME", c.appID)

	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ebay finding: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebay finding: status %d", resp.StatusCode)
	}

	var out findingResponse
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ebay finding: decode response: %w", err)
	}
	if !strings.EqualFold(out.Ack, "success") {
		return nil, fmt.Errorf("ebay finding: ack=%s", out.Ack)
	}

	return &out, nil
}

// FetchSold grabs completed fixed-price listings up to maxItems.
// Uses the (deprecated but operational) Finding API findCompletedItems operation.
func (c *findingClient) FetchSold(ctx context.Context, query string, maxItems int) ([]domain.RawListing, error) {
	if maxItems <= 0 {
		return []domain.RawListing{}, nil
	}

	const pageSize = 100
	listings := make([]domain.RawListing, 0, maxItems)

	for page := 1; len(listings) < maxItems; page++ {
		remaining := maxItems - len(listings)
		limit := pageSize
		if remaining < pageSize {
			limit = remaining
		}

		result, err := c.fetchPage(ctx, query, page, limit)
		if err != nil {
			return nil, err
		}
		if len(result.SearchResult.Items) == 0 {
			break
		}

		now := time.Now().UTC()
		for _, item := range result.SearchResult.Items {
			if len(listings) >= maxItems {
				break
			}

			priceCents, err := parsePriceCents(item.SellingStatus.CurrentPrice.Value)
			if err != nil {
				continue
			}

			listings = append(listings, domain.RawListing{
				ID:              uuid.New(),
				SourceID:        c.sourceID,
				SourceListingID: item.ItemID,
				Title:           item.Title,
				ConditionRaw:    item.Condition.DisplayName,
				PriceCents:      priceCents,
				Currency:        item.SellingStatus.CurrentPrice.CurrencyID,
				ListingType:     domain.ListingTypeSold,
				URL:             item.ViewItemURL,
				FetchedAt:       now,
			})
		}

		if page >= result.Pagination.TotalPages {
			break
		}
	}

	return listings, nil
}

