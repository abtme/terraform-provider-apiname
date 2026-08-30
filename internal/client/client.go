package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

// debugHTTP reports whether APINAME_DEBUG_HTTP is set, enabling verbose
// request/response logging to stderr. The API key lives only in a header
// here (never logged), but body/URL are redacted too as defense-in-depth.
func debugHTTP() bool {
	return os.Getenv("APINAME_DEBUG_HTTP") != ""
}

func (c *Client) redact(s string) string {
	if c.APIKey != "" {
		s = strings.ReplaceAll(s, c.APIKey, "[redacted]")
	}
	return s
}

// request performs an HTTP call against the DomainNameAPI REST gateway and
// decodes a successful JSON response into target (if non-nil).
func (c *Client) request(method, endpoint string, query url.Values, body interface{}, target interface{}) error {
	reqURL := c.BaseURL + "/" + strings.TrimLeft(endpoint, "/")
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	var bodyForLog string
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
		bodyForLog = string(b)
	}
	if debugHTTP() {
		fmt.Fprintf(os.Stderr, "APINAME_DEBUG_HTTP: %s %s body=%q\n", method, c.redact(reqURL), c.redact(bodyForLog))
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
		if debugHTTP() {
			fmt.Fprintf(os.Stderr, "APINAME_DEBUG_HTTP: request failed: %v\n", err)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if debugHTTP() {
		fmt.Fprintf(os.Stderr, "APINAME_DEBUG_HTTP: response status=%s\n", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if debugHTTP() {
		fmt.Fprintf(os.Stderr, "APINAME_DEBUG_HTTP: response body=%q\n", c.redact(string(respBody)))
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
