package proxmox

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
	"github.com/eblackrps/viaduct/internal/telemetry"
)

type apiEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type ticketResponse struct {
	Ticket              string `json:"ticket"`
	CSRFPreventionToken string `json:"CSRFPreventionToken"`
}

// ProxmoxClient is a lightweight HTTP client for the Proxmox VE API.
type ProxmoxClient struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	authTicket string
	csrfToken  string
	requestID  string
}

// NewProxmoxClient builds a Proxmox API client for the provided address.
func NewProxmoxClient(address string, insecure bool, requestID ...string) *ProxmoxClient {
	baseURL := normalizeBaseURL(address)
	transport := &http.Transport{
		// #nosec G402 -- operators can explicitly opt into insecure TLS for lab and self-signed Proxmox endpoints.
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure},
	}
	propagatedRequestID := ""
	if len(requestID) > 0 {
		propagatedRequestID = strings.TrimSpace(requestID[0])
	}

	return &ProxmoxClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: telemetry.HTTPClientTransport(transport, "proxmox"),
		},
		requestID: propagatedRequestID,
	}
}

// Authenticate exchanges a username and password for a Proxmox API ticket.
func (c *ProxmoxClient) Authenticate(ctx context.Context, username, password string) error {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/access/ticket", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("proxmox: build auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.requestID != "" {
		req.Header.Set(connectors.RequestIDHeader, c.requestID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("proxmox: authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("proxmox: authenticate status %d: %w", resp.StatusCode, readErr)
		}

		return fmt.Errorf("proxmox: authenticate status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("proxmox: read auth response: %w", err)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("proxmox: decode auth envelope: %w", err)
	}

	var ticket ticketResponse
	if err := json.Unmarshal(envelope.Data, &ticket); err != nil {
		return fmt.Errorf("proxmox: decode auth payload: %w", err)
	}

	c.authTicket = ticket.Ticket
	c.csrfToken = ticket.CSRFPreventionToken

	return nil
}

// AuthenticateToken configures token-based authentication for subsequent requests.
func (c *ProxmoxClient) AuthenticateToken(tokenID, tokenSecret string) {
	c.authToken = fmt.Sprintf("%s=%s", tokenID, tokenSecret)
	c.authTicket = ""
	c.csrfToken = ""
}

// Get executes an authenticated GET request against the Proxmox API.
func (c *ProxmoxClient) Get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL(endpoint), nil)
	if err != nil {
		return nil, fmt.Errorf("proxmox: build GET request: %w", err)
	}

	c.addAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxmox: GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("proxmox: read GET %s response: %w", endpoint, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("proxmox: GET %s status %d: %s", endpoint, resp.StatusCode, bytes.TrimSpace(body))
	}

	return body, nil
}

func (c *ProxmoxClient) decode(ctx context.Context, endpoint string, out any) error {
	body, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("proxmox: decode %s envelope: %w", endpoint, err)
	}

	if len(envelope.Data) == 0 {
		return nil
	}

	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("proxmox: decode %s payload: %w", endpoint, err)
	}

	return nil
}

func (c *ProxmoxClient) buildURL(endpoint string) string {
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func (c *ProxmoxClient) addAuthHeaders(req *http.Request) {
	if c.requestID != "" {
		req.Header.Set(connectors.RequestIDHeader, c.requestID)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "PVEAPIToken="+c.authToken)
	}

	if c.authTicket != "" {
		req.Header.Set("Cookie", "PVEAuthCookie="+c.authTicket)
	}

	if c.csrfToken != "" {
		req.Header.Set("CSRFPreventionToken", c.csrfToken)
	}
}

func normalizeBaseURL(address string) string {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		parsed, err := url.Parse(address)
		if err == nil {
			parsed.Path = path.Join(parsed.Path, "/api2/json")
			return strings.TrimRight(parsed.String(), "/")
		}
	}

	address = strings.TrimRight(address, "/")
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return strings.TrimRight(address, "/") + "/api2/json"
	}

	if parsed.Port() == "" {
		parsed.Host = parsed.Host + ":8006"
	}

	parsed.Path = path.Join(parsed.Path, "/api2/json")

	return strings.TrimRight(parsed.String(), "/")
}
