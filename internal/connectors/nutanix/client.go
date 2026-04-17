package nutanix

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
)

// PrismClient is a lightweight Prism Central v3 REST client.
type PrismClient struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	requestID  string
}

// NewPrismClient creates a Prism Central client for the supplied endpoint.
func NewPrismClient(address, username, password string, insecure bool, requestID ...string) *PrismClient {
	address = strings.TrimSpace(address)
	if address == "" {
		address = "https://localhost:9440"
	}
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "https://" + address
	}
	propagatedRequestID := ""
	if len(requestID) > 0 {
		propagatedRequestID = strings.TrimSpace(requestID[0])
	}

	return &PrismClient{
		baseURL: strings.TrimRight(address, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				// #nosec G402 -- operators can explicitly opt into insecure TLS for lab and self-signed Prism endpoints.
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure},
			},
		},
		username:  username,
		password:  password,
		requestID: propagatedRequestID,
	}
}

// Post performs an authenticated POST request.
func (c *PrismClient) Post(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	body, err := c.do(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("nutanix: POST %s: %w", endpoint, err)
	}
	return body, nil
}

// Get performs an authenticated GET request.
func (c *PrismClient) Get(ctx context.Context, endpoint string) ([]byte, error) {
	body, err := c.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("nutanix: GET %s: %w", endpoint, err)
	}
	return body, nil
}

// ListAll pages through a Prism v3 list endpoint and returns all entities.
func (c *PrismClient) ListAll(ctx context.Context, endpoint string, payload map[string]interface{}) ([]map[string]interface{}, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	all := make([]map[string]interface{}, 0)
	offset := 0
	length := intValue(payload["length"])
	if length <= 0 {
		length = 50
	}

	for {
		pagePayload := clonePayload(payload)
		pagePayload["offset"] = offset
		pagePayload["length"] = length

		body, err := c.Post(ctx, endpoint, pagePayload)
		if err != nil {
			return nil, err
		}

		var response struct {
			Entities []map[string]interface{} `json:"entities"`
			Metadata struct {
				Length       int `json:"length"`
				Offset       int `json:"offset"`
				TotalMatches int `json:"total_matches"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("nutanix: decode %s response: %w", endpoint, err)
		}

		all = append(all, response.Entities...)
		offset += response.Metadata.Length
		if response.Metadata.Length == 0 || offset >= response.Metadata.TotalMatches {
			break
		}
	}

	return all, nil
}

func (c *PrismClient) do(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	if c.requestID != "" {
		req.Header.Set(connectors.RequestIDHeader, c.requestID)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func clonePayload(input map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
