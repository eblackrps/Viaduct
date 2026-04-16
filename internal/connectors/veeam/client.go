package veeam

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
)

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// VeeamClient is a lightweight REST client for Veeam Backup & Replication.
type VeeamClient struct {
	baseURL     string
	httpClient  *http.Client
	accessToken string
	username    string
	password    string
	requestID   string
}

// NewVeeamClient creates a new Veeam REST client for the supplied address.
func NewVeeamClient(address string, insecure bool, requestID ...string) *VeeamClient {
	propagatedRequestID := ""
	if len(requestID) > 0 {
		propagatedRequestID = strings.TrimSpace(requestID[0])
	}
	return &VeeamClient{
		baseURL: normalizeBaseURL(address),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure},
			},
		},
		requestID: propagatedRequestID,
	}
}

// Authenticate exchanges a username and password for a Veeam access token.
func (c *VeeamClient) Authenticate(ctx context.Context, username, password string) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("veeam: build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.requestID != "" {
		req.Header.Set(connectors.RequestIDHeader, c.requestID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("veeam: authenticate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("veeam: read auth response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("veeam: authenticate status %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var token oauthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("veeam: decode auth response: %w", err)
	}
	if token.AccessToken == "" {
		return fmt.Errorf("veeam: authenticate: access token missing")
	}

	c.accessToken = token.AccessToken
	c.username = username
	c.password = password
	return nil
}

// Get performs an authenticated GET request and refreshes the token on 401 responses.
func (c *VeeamClient) Get(ctx context.Context, endpoint string) ([]byte, error) {
	body, status, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusUnauthorized {
		return body, nil
	}
	if c.username == "" || c.password == "" {
		return nil, fmt.Errorf("veeam: GET %s status %d", endpoint, status)
	}

	if err := c.Authenticate(ctx, c.username, c.password); err != nil {
		return nil, fmt.Errorf("veeam: refresh token: %w", err)
	}

	body, _, err = c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Post performs an authenticated POST request and refreshes the token on 401 responses.
func (c *VeeamClient) Post(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	return c.doJSONWithRetry(ctx, http.MethodPost, endpoint, payload)
}

// Put performs an authenticated PUT request and refreshes the token on 401 responses.
func (c *VeeamClient) Put(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	return c.doJSONWithRetry(ctx, http.MethodPut, endpoint, payload)
}

// Delete performs an authenticated DELETE request and refreshes the token on 401 responses.
func (c *VeeamClient) Delete(ctx context.Context, endpoint string) error {
	_, err := c.doJSONWithRetry(ctx, http.MethodDelete, endpoint, nil)
	return err
}

func (c *VeeamClient) doJSONWithRetry(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	body, status, err := c.doRequest(ctx, method, endpoint, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusUnauthorized {
		return body, nil
	}
	if c.username == "" || c.password == "" {
		return nil, fmt.Errorf("veeam: %s %s status %d", method, endpoint, status)
	}

	if err := c.Authenticate(ctx, c.username, c.password); err != nil {
		return nil, fmt.Errorf("veeam: refresh token: %w", err)
	}

	body, _, err = c.doRequest(ctx, method, endpoint, payload)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *VeeamClient) doRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, int, error) {
	bodyReader, err := c.newBodyReader(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("veeam: encode %s %s payload: %w", method, endpoint, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(endpoint), bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("veeam: build %s %s: %w", method, endpoint, err)
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	if c.requestID != "" {
		req.Header.Set(connectors.RequestIDHeader, c.requestID)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("veeam: %s %s: %w", method, endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("veeam: read %s %s response: %w", method, endpoint, err)
	}
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
		return nil, resp.StatusCode, fmt.Errorf("veeam: %s %s status %d: %s", method, endpoint, resp.StatusCode, bytes.TrimSpace(body))
	}

	return body, resp.StatusCode, nil
}

func (c *VeeamClient) newBodyReader(payload interface{}) (io.Reader, error) {
	if payload == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(encoded), nil
}

func (c *VeeamClient) buildURL(endpoint string) string {
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func normalizeBaseURL(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return "https://localhost:9419/api"
	}

	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "https://" + address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return strings.TrimRight(address, "/") + "/api"
	}
	if parsed.Port() == "" {
		parsed.Host = parsed.Host + ":9419"
	}
	parsed.Path = path.Join(parsed.Path, "/api")
	return strings.TrimRight(parsed.String(), "/")
}
