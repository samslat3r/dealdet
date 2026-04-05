package ebay

/*
 *
 * Oauth token cache for ebay. access/refresh. Makes sure everything can get a valid (non-expired) bearer token.
 * Refresh logic is here too
 */

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// we only need to use one goroutine to refresh the ebay oauth2 token
// mutex... yeah, goroutine refreshes when the token expires

type tokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// get a valid token, refresh if within 5min of expiration

func (tc *tokenCache) get(ctx context.Context, appID, certID, env string) (string, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	//return cached token if valid with 5min buffer
	if tc.token != "" && time.Now().Before(tc.expiresAt.Add(-5*time.Minute)) {
		return tc.token, nil
	}
	tokenURL := "https://api.ebay.com/identity/v1/oauth2/token"
	if env == "sandbox" {
		tokenURL = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	}

	// key managemenet later
	creds := base64.StdEncoding.EncodeToString([]byte(appID + ":" + certID))

	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"https://api.ebay.com/oauth/api_scope"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("ebay auth: build request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ebay auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ebay auth: status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ebay auth: decode %w", err)
	}

	tc.token = result.AccessToken
	tc.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return tc.token, nil

}
