package hyperv

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/masterzen/winrm"
)

// WinRMClient executes PowerShell commands against a remote Hyper-V host.
type WinRMClient struct {
	host     string
	port     int
	username string
	password string
	useHTTPS bool
}

// NewWinRMClient creates a WinRM client from a generic connector config.
func NewWinRMClient(config connectors.Config) *WinRMClient {
	host := strings.TrimSpace(config.Address)
	useHTTPS := true
	port := config.Port

	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil {
			host = parsed.Hostname()
			useHTTPS = parsed.Scheme != "http"
			if parsed.Port() != "" {
				if parsedPort, err := strconv.Atoi(parsed.Port()); err == nil {
					port = parsedPort
				}
			}
		}
	}

	if port == 0 {
		if useHTTPS {
			port = 5986
		} else {
			port = 5985
		}
	}

	return &WinRMClient{
		host:     host,
		port:     port,
		username: config.Username,
		password: config.Password,
		useHTTPS: useHTTPS,
	}
}

// Execute runs a PowerShell command remotely and returns stdout.
func (c *WinRMClient) Execute(ctx context.Context, psCommand string) (string, error) {
	endpoint := winrm.NewEndpoint(c.host, c.port, c.useHTTPS, true, nil, nil, nil, 0)
	client, err := winrm.NewClient(endpoint, c.username, c.password)
	if err != nil {
		return "", fmt.Errorf("hyperv: create WinRM client: %w", err)
	}

	stdout, stderr, _, err := client.RunPSWithContext(ctx, psCommand)
	if err != nil {
		return "", fmt.Errorf("hyperv: execute PowerShell: %w: %s", err, strings.TrimSpace(stderr))
	}

	return stdout, nil
}
