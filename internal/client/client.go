package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the DomainNameAPI ("ApiName") reseller API client.
type Client struct {
	BaseURL    string
	ResellerID string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new DomainNameAPI client.
func NewClient(baseURL, resellerID, apiKey string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		ResellerID: resellerID,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// domainInfoResponse is the relevant subset of GET domains/info's response.
type domainInfoResponse struct {
	DomainName  string   `json:"domainName"`
	NameServers []string `json:"nameServers"`
}

// errorResponse covers both the flat and {"error": {...}}-nested shapes the
// gateway returns on non-2xx responses.
type errorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Details string `json:"details"`
	Error   struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Details string `json:"details"`
	} `json:"error"`
}

// request performs an HTTP call against the DomainNameAPI REST gateway and
// decodes a successful JSON response into target (if non-nil).
func (c *Client) request(method, endpoint string, query url.Values, body interface{}, target interface{}) error {
	reqURL := c.BaseURL + "/" + strings.TrimLeft(endpoint, "/")
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-KEY", c.APIKey)
	req.Header.Set("__reseller", c.ResellerID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("DomainNameAPI error: %s", parseError(resp.StatusCode, respBody))
	}

	if target != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("parsing JSON response: %w (body: %s)", err, string(respBody))
		}
	}

	return nil
}

// parseError extracts a human-readable message from a non-2xx response body,
// falling back to a synthesised message when the body is empty or unparsable
// (e.g. a 429/502/503 from the gateway with no JSON body).
func parseError(status int, body []byte) string {
	var e errorResponse
	if err := json.Unmarshal(body, &e); err == nil {
		msg := e.Message
		if msg == "" {
			msg = e.Error.Message
		}
		if msg != "" {
			return fmt.Sprintf("HTTP %d: %s", status, msg)
		}
	}
	if len(body) > 0 {
		return fmt.Sprintf("HTTP %d: %s", status, string(body))
	}
	return fmt.Sprintf("HTTP %d", status)
}

// ---------------------------------------------------------------------------
// GetNameservers
// ---------------------------------------------------------------------------

// GetNameservers fetches the current nameservers for the given domain name.
func (c *Client) GetNameservers(domain string) ([]string, error) {
	query := url.Values{}
	query.Set("DomainName", domain)

	var result domainInfoResponse
	if err := c.request(http.MethodGet, "domains/info", query, nil, &result); err != nil {
		return nil, err
	}

	var nameservers []string
	for _, ns := range result.NameServers {
		if ns != "" {
			nameservers = append(nameservers, ns)
		}
	}
	return nameservers, nil
}

// ---------------------------------------------------------------------------
// ModifyNameservers
// ---------------------------------------------------------------------------

// ModifyNameservers updates the nameservers for the given domain name.
func (c *Client) ModifyNameservers(domain string, nameservers []string) error {
	payload := map[string]interface{}{
		"domainName":  domain,
		"nameServers": nameservers,
	}
	return c.request(http.MethodPut, "domains/dns/name-server", nil, payload, nil)
}
