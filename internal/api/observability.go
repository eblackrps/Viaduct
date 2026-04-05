package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type metricKey struct {
	method string
	path   string
	status int
}

type metricValue struct {
	count          int
	durationMicros int64
}

type apiMetrics struct {
	mu       sync.Mutex
	inFlight int
	series   map[metricKey]metricValue
}

func newAPIMetrics() *apiMetrics {
	return &apiMetrics{
		series: make(map[metricKey]metricValue),
	}
}

func (m *apiMetrics) record(method, path string, status int, duration time.Duration) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey{
		method: method,
		path:   normalizeMetricsPath(path),
		status: status,
	}
	value := m.series[key]
	value.count++
	value.durationMicros += duration.Microseconds()
	m.series[key] = value
}

func (m *apiMetrics) startRequest() func(string, string, int, time.Duration) {
	if m == nil {
		return func(string, string, int, time.Duration) {}
	}

	m.mu.Lock()
	m.inFlight++
	m.mu.Unlock()

	return func(method, path string, status int, duration time.Duration) {
		m.mu.Lock()
		m.inFlight--
		m.mu.Unlock()
		m.record(method, path, status, duration)
	}
}

func (m *apiMetrics) render() string {
	if m == nil {
		return ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	lines := []string{
		"# HELP viaduct_http_requests_total Total HTTP requests served by the Viaduct API.",
		"# TYPE viaduct_http_requests_total counter",
		"# HELP viaduct_http_request_duration_seconds_sum Total HTTP request duration by route.",
		"# TYPE viaduct_http_request_duration_seconds_sum counter",
		"# HELP viaduct_http_requests_in_flight In-flight HTTP requests.",
		"# TYPE viaduct_http_requests_in_flight gauge",
		fmt.Sprintf("viaduct_http_requests_in_flight %d", m.inFlight),
	}

	keys := make([]metricKey, 0, len(m.series))
	for key := range m.series {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].path == keys[j].path {
			if keys[i].method == keys[j].method {
				return keys[i].status < keys[j].status
			}
			return keys[i].method < keys[j].method
		}
		return keys[i].path < keys[j].path
	})

	for _, key := range keys {
		value := m.series[key]
		labels := fmt.Sprintf(`method=%q,path=%q,status=%q`, key.method, key.path, strconv.Itoa(key.status))
		lines = append(lines, fmt.Sprintf("viaduct_http_requests_total{%s} %d", labels, value.count))
		lines = append(lines, fmt.Sprintf("viaduct_http_request_duration_seconds_sum{%s} %.6f", labels, float64(value.durationMicros)/1_000_000))
	}

	return strings.Join(lines, "\n") + "\n"
}

type tenantRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]rateBucket
}

type rateBucket struct {
	windowStart time.Time
	count       int
}

func newTenantRateLimiter(limit int, window time.Duration) *tenantRateLimiter {
	if limit <= 0 || window <= 0 {
		return nil
	}
	return &tenantRateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]rateBucket),
	}
}

func (l *tenantRateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= l.window {
		bucket = rateBucket{windowStart: now, count: 0}
	}
	if bucket.count >= l.limit {
		retryAfter := l.window - now.Sub(bucket.windowStart)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	bucket.count++
	l.buckets[key] = bucket
	return true, 0
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(s.metrics.render()))
}

func (s *Server) withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		startedAt := time.Now()
		done := s.metrics.startRequest()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		recorder.Header().Set(requestIDHeader, requestID)

		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(recorder, r.WithContext(ctx))

		duration := time.Since(startedAt)
		done(r.Method, r.URL.Path, recorder.status, duration)
		log.Printf(
			"component=api request_id=%s method=%s path=%s status=%d duration_ms=%d tenant=%s",
			requestID,
			r.Method,
			r.URL.Path,
			recorder.status,
			duration.Milliseconds(),
			store.TenantIDFromContext(ctx),
		)
	})
}

// TenantRateLimitMiddleware enforces a simple per-tenant request budget.
func TenantRateLimitMiddleware(limiter *tenantRateLimiter, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := store.TenantIDFromContext(r.Context())
		allowed, retryAfter := limiter.allow(tenantID, time.Now().UTC())
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			http.Error(w, "tenant rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestIDFromContext returns the request identifier attached by API observability middleware.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}

func normalizeMetricsPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/migrations/"):
		trimmed := strings.Trim(path, "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 5 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id", parts[4]}, "/")
		}
		if len(parts) >= 4 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id"}, "/")
		}
	case strings.HasPrefix(path, "/api/v1/snapshots/"):
		return "/api/v1/snapshots/:id"
	case strings.HasPrefix(path, "/api/v1/admin/tenants/"):
		return "/api/v1/admin/tenants/:id"
	case strings.HasPrefix(path, "/api/v1/reports/"):
		trimmed := strings.Trim(path, "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 4 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], parts[3]}, "/")
		}
	}
	return path
}
