package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

type connectorCircuitState string

const (
	connectorCircuitClosed   connectorCircuitState = "closed"
	connectorCircuitOpen     connectorCircuitState = "open"
	connectorCircuitHalfOpen connectorCircuitState = "half-open"
)

type connectorCircuitConfig struct {
	FailureThreshold int
	FailureWindow    time.Duration
	OpenDuration     time.Duration
}

type connectorCircuitRegistry struct {
	mu       sync.Mutex
	config   connectorCircuitConfig
	circuits map[string]*connectorCircuit
}

type connectorCircuit struct {
	key              string
	platform         models.Platform
	address          string
	state            connectorCircuitState
	failures         []time.Time
	openUntil        time.Time
	lastFailure      string
	lastFailureAt    time.Time
	lastStateChanged time.Time
}

type connectorCircuitSnapshot struct {
	Endpoint           string                `json:"endpoint"`
	Platform           models.Platform       `json:"platform"`
	Address            string                `json:"address"`
	State              connectorCircuitState `json:"state"`
	FailureCount       int                   `json:"failure_count"`
	LastFailure        string                `json:"last_failure,omitempty"`
	LastFailureAt      time.Time             `json:"last_failure_at,omitempty"`
	RetryAfterSeconds  int                   `json:"retry_after_seconds,omitempty"`
	OpenedUntil        time.Time             `json:"opened_until,omitempty"`
	LastStateChangedAt time.Time             `json:"last_state_changed_at,omitempty"`
}

// ConnectorUnavailableError reports an open connector circuit and the suggested retry window.
type ConnectorUnavailableError struct {
	Platform   models.Platform
	Address    string
	Endpoint   string
	RetryAfter time.Duration
}

func (e *ConnectorUnavailableError) Error() string {
	if e == nil {
		return "connector circuit is unavailable"
	}
	if e.RetryAfter <= 0 {
		return fmt.Sprintf("connector %s is temporarily unavailable", e.Endpoint)
	}
	return fmt.Sprintf("connector %s is temporarily unavailable; retry after %ds", e.Endpoint, int(e.RetryAfter.Seconds())+1)
}

func newConnectorCircuitRegistry(config connectorCircuitConfig) *connectorCircuitRegistry {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.FailureWindow <= 0 {
		config.FailureWindow = time.Minute
	}
	if config.OpenDuration <= 0 {
		config.OpenDuration = time.Minute
	}
	return &connectorCircuitRegistry{
		config:   config,
		circuits: make(map[string]*connectorCircuit),
	}
}

func loadConnectorCircuitConfig() connectorCircuitConfig {
	return connectorCircuitConfig{
		FailureThreshold: intEnvValue("VIADUCT_CONNECTOR_CIRCUIT_FAILURES", 5),
		FailureWindow:    durationEnv("VIADUCT_CONNECTOR_CIRCUIT_WINDOW", time.Minute),
		OpenDuration:     durationEnv("VIADUCT_CONNECTOR_CIRCUIT_OPEN_DURATION", time.Minute),
	}
}

func intEnvValue(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (r *connectorCircuitRegistry) Wrap(platform models.Platform, address string, connector connectors.Connector) connectors.Connector {
	if r == nil || connector == nil {
		return connector
	}
	key, normalizedAddress := connectorEndpointKey(platform, address)
	return &circuitBreakerConnector{
		registry:  r,
		key:       key,
		platform:  platform,
		address:   normalizedAddress,
		connector: connector,
	}
}

func (r *connectorCircuitRegistry) CheckAvailability(platform models.Platform, address string) error {
	if r == nil {
		return nil
	}
	key, normalizedAddress := connectorEndpointKey(platform, address)
	_, err := r.allow(key, platform, normalizedAddress)
	return err
}

func (r *connectorCircuitRegistry) allow(key string, platform models.Platform, address string) (bool, error) {
	if r == nil {
		return false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	circuit := r.circuitLocked(key, platform, address)
	now := time.Now().UTC()
	circuit.failures = r.pruneFailures(circuit.failures, now)

	switch circuit.state {
	case connectorCircuitOpen:
		if now.Before(circuit.openUntil) {
			return false, &ConnectorUnavailableError{
				Platform:   platform,
				Address:    address,
				Endpoint:   key,
				RetryAfter: time.Until(circuit.openUntil),
			}
		}
		r.transitionLocked(circuit, connectorCircuitHalfOpen, now, "retry window elapsed")
		return true, nil
	case connectorCircuitHalfOpen:
		return true, nil
	default:
		return false, nil
	}
}

func (r *connectorCircuitRegistry) recordSuccess(key string, platform models.Platform, address string, probe bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	circuit := r.circuitLocked(key, platform, address)
	now := time.Now().UTC()
	circuit.failures = nil
	circuit.lastFailure = ""
	circuit.lastFailureAt = time.Time{}
	if probe || circuit.state != connectorCircuitClosed {
		r.transitionLocked(circuit, connectorCircuitClosed, now, "downstream call succeeded")
	}
}

func (r *connectorCircuitRegistry) recordFailure(key string, platform models.Platform, address string, err error) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	circuit := r.circuitLocked(key, platform, address)
	now := time.Now().UTC()
	circuit.failures = append(r.pruneFailures(circuit.failures, now), now)
	circuit.lastFailure = strings.TrimSpace(err.Error())
	circuit.lastFailureAt = now

	if circuit.state == connectorCircuitHalfOpen || len(circuit.failures) >= r.config.FailureThreshold {
		circuit.openUntil = now.Add(r.config.OpenDuration)
		r.transitionLocked(circuit, connectorCircuitOpen, now, "failure threshold reached")
	}
}

func (r *connectorCircuitRegistry) snapshots() []connectorCircuitSnapshot {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	snapshots := make([]connectorCircuitSnapshot, 0, len(r.circuits))
	for _, circuit := range r.circuits {
		circuit.failures = r.pruneFailures(circuit.failures, now)
		snapshot := connectorCircuitSnapshot{
			Endpoint:           circuit.key,
			Platform:           circuit.platform,
			Address:            circuit.address,
			State:              circuit.state,
			FailureCount:       len(circuit.failures),
			LastFailure:        circuit.lastFailure,
			LastFailureAt:      circuit.lastFailureAt,
			LastStateChangedAt: circuit.lastStateChanged,
		}
		if circuit.state == connectorCircuitOpen && now.Before(circuit.openUntil) {
			snapshot.OpenedUntil = circuit.openUntil
			snapshot.RetryAfterSeconds = int(time.Until(circuit.openUntil).Seconds()) + 1
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func (r *connectorCircuitRegistry) circuitLocked(key string, platform models.Platform, address string) *connectorCircuit {
	if circuit, ok := r.circuits[key]; ok {
		return circuit
	}
	circuit := &connectorCircuit{
		key:      key,
		platform: platform,
		address:  address,
		state:    connectorCircuitClosed,
	}
	r.circuits[key] = circuit
	return circuit
}

func (r *connectorCircuitRegistry) pruneFailures(failures []time.Time, now time.Time) []time.Time {
	if len(failures) == 0 {
		return nil
	}
	cutoff := now.Add(-r.config.FailureWindow)
	kept := failures[:0]
	for _, failure := range failures {
		if failure.After(cutoff) {
			kept = append(kept, failure)
		}
	}
	return kept
}

func (r *connectorCircuitRegistry) transitionLocked(circuit *connectorCircuit, next connectorCircuitState, now time.Time, reason string) {
	if circuit == nil || circuit.state == next {
		return
	}
	previous := circuit.state
	circuit.state = next
	circuit.lastStateChanged = now
	packageLogger.Warn(
		"connector circuit state changed",
		"endpoint", circuit.key,
		"platform", circuit.platform,
		"state", next,
		"previous_state", previous,
		"reason", reason,
	)
}

func connectorEndpointKey(platform models.Platform, address string) (string, string) {
	normalizedAddress := strings.ToLower(strings.TrimSpace(address))
	if normalizedAddress == "" {
		normalizedAddress = "unknown"
	}
	return string(platform) + ":" + normalizedAddress, normalizedAddress
}

func connectorUnavailableError(err error) (*ConnectorUnavailableError, bool) {
	var target *ConnectorUnavailableError
	if !errors.As(err, &target) {
		return nil, false
	}
	return target, true
}

func writeConnectorUnavailable(w http.ResponseWriter, r *http.Request, err error) bool {
	unavailable, ok := connectorUnavailableError(err)
	if !ok {
		return false
	}
	if unavailable.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(unavailable.RetryAfter.Seconds())+1))
	}
	writeAPIError(w, r, http.StatusServiceUnavailable, "connector_unavailable", unavailable.Error(), apiErrorOptions{
		Retryable: true,
		Details: map[string]any{
			"endpoint":            unavailable.Endpoint,
			"platform":            unavailable.Platform,
			"address":             unavailable.Address,
			"retry_after_seconds": int(unavailable.RetryAfter.Seconds()) + 1,
		},
	})
	return true
}

type circuitBreakerConnector struct {
	registry  *connectorCircuitRegistry
	key       string
	platform  models.Platform
	address   string
	connector connectors.Connector
	probeMode bool
}

func (c *circuitBreakerConnector) Connect(ctx context.Context) error {
	probe, err := c.registry.allow(c.key, c.platform, c.address)
	if err != nil {
		return err
	}
	if probe {
		c.probeMode = true
	}
	if err := c.connector.Connect(ctx); err != nil {
		c.registry.recordFailure(c.key, c.platform, c.address, err)
		return err
	}
	if !c.probeMode {
		c.registry.recordSuccess(c.key, c.platform, c.address, false)
	}
	return nil
}

func (c *circuitBreakerConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	probe, err := c.registry.allow(c.key, c.platform, c.address)
	if err != nil {
		return nil, err
	}
	if probe {
		c.probeMode = true
	}
	result, err := c.connector.Discover(ctx)
	if err != nil {
		c.registry.recordFailure(c.key, c.platform, c.address, err)
		return nil, err
	}
	c.registry.recordSuccess(c.key, c.platform, c.address, c.probeMode)
	c.probeMode = false
	return result, nil
}

func (c *circuitBreakerConnector) Platform() models.Platform {
	return c.connector.Platform()
}

func (c *circuitBreakerConnector) Close() error {
	return c.connector.Close()
}

var _ connectors.Connector = (*circuitBreakerConnector)(nil)

func contextWithConnectorRequestID(ctx context.Context, requestID string) context.Context {
	return ContextWithRequestID(ctx, requestID)
}
