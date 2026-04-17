package api

import (
	"context"
	"fmt"
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
type requestScopeContextKey struct{}

type requestScope struct {
	requestID  string
	tenantID   string
	authMethod string
}

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

	func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.inFlight++
	}()

	return func(method, path string, status int, duration time.Duration) {
		func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.inFlight--
		}()
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
	limit       int
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

func (l *tenantRateLimiter) allow(key string, now time.Time, overrideLimit int) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	effectiveLimit := l.limit
	if overrideLimit > 0 {
		effectiveLimit = overrideLimit
	}

	bucket := l.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= l.window || bucket.limit != effectiveLimit {
		bucket = rateBucket{windowStart: now, count: 0, limit: effectiveLimit}
	}
	if bucket.count >= effectiveLimit {
		retryAfter := l.window - now.Sub(bucket.windowStart)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	bucket.count++
	bucket.limit = effectiveLimit
	l.buckets[key] = bucket
	return true, 0
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	metrics := s.metrics.render() + s.renderOperationalMetrics(r.Context())
	_, _ = w.Write([]byte(metrics))
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

		scope := &requestScope{
			requestID: requestID,
			tenantID:  store.TenantIDFromContext(r.Context()),
		}
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		ctx = context.WithValue(ctx, requestScopeContextKey{}, scope)
		next.ServeHTTP(recorder, r.WithContext(ctx))

		duration := time.Since(startedAt)
		done(r.Method, r.URL.Path, recorder.status, duration)
		packageLogger.Info(
			"http request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", duration.Milliseconds(),
			"tenant_id", scope.tenantID,
			"auth_method", scope.authMethod,
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
		overrideLimit := 0
		if principal, err := RequirePrincipal(r.Context()); err == nil {
			overrideLimit = principal.Tenant.Quotas.RequestsPerMinute
		}
		allowed, retryAfter := limiter.allow(tenantID, time.Now().UTC(), overrideLimit)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			writeAPIError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded", "tenant rate limit exceeded", apiErrorOptions{
				Retryable: true,
				Details: map[string]any{
					"retry_after_seconds": int(retryAfter.Seconds()) + 1,
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientRateLimitMiddleware enforces a stricter pre-auth rate limit keyed by client IP.
func ClientRateLimitMiddleware(limiter *tenantRateLimiter, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
		if clientIP != "" {
			clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
		}
		if clientIP == "" {
			clientIP = strings.TrimSpace(r.Header.Get("X-Real-IP"))
		}
		if clientIP == "" {
			clientIP = strings.TrimSpace(r.RemoteAddr)
		}

		allowed, retryAfter := limiter.allow(clientIP, time.Now().UTC(), 0)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			writeAPIError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded", "authentication rate limit exceeded", apiErrorOptions{
				Retryable: true,
				Details: map[string]any{
					"retry_after_seconds": int(retryAfter.Seconds()) + 1,
				},
			})
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

// ContextWithRequestID attaches a request identifier to a context for downstream correlation.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestScopeFromContext(ctx context.Context) *requestScope {
	if ctx == nil {
		return nil
	}
	scope, _ := ctx.Value(requestScopeContextKey{}).(*requestScope)
	return scope
}

func normalizeMetricsPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/migrations/"), strings.HasPrefix(path, "/api/v2/migrations/"):
		trimmed := strings.Trim(path, "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 5 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id", parts[4]}, "/")
		}
		if len(parts) >= 4 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id"}, "/")
		}
	case strings.HasPrefix(path, "/api/v1/snapshots/"), strings.HasPrefix(path, "/api/v2/snapshots/"):
		trimmed := strings.Trim(path, "/")
		parts := strings.Split(trimmed, "/")
		return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id"}, "/")
	case strings.HasPrefix(path, "/api/v1/workspaces/"), strings.HasPrefix(path, "/api/v2/workspaces/"):
		trimmed := strings.Trim(path, "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 6 && parts[4] == "jobs" {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id", parts[4], ":job_id"}, "/")
		}
		if len(parts) >= 6 && parts[4] == "reports" {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id", parts[4], parts[5]}, "/")
		}
		if len(parts) >= 5 {
			return "/" + strings.Join([]string{parts[0], parts[1], parts[2], ":id", parts[4]}, "/")
		}
		return "/api/v1/workspaces/:id"
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

func (s *Server) renderOperationalMetrics(ctx context.Context) string {
	if s == nil || s.store == nil {
		return ""
	}

	lines := []string{
		"# HELP viaduct_tenant_snapshots Total persisted discovery snapshots per tenant.",
		"# TYPE viaduct_tenant_snapshots gauge",
		"# HELP viaduct_tenant_service_accounts Total service accounts configured per tenant.",
		"# TYPE viaduct_tenant_service_accounts gauge",
		"# HELP viaduct_tenant_migrations Total persisted migrations per tenant and phase.",
		"# TYPE viaduct_tenant_migrations gauge",
		"# HELP viaduct_tenant_quota_remaining Remaining tenant quota budget by resource.",
		"# TYPE viaduct_tenant_quota_remaining gauge",
	}

	if provider, ok := s.store.(store.DiagnosticsProvider); ok {
		if diagnostics, err := provider.Diagnostics(ctx); err == nil {
			lines = append(lines,
				"# HELP viaduct_store_info Store backend metadata.",
				"# TYPE viaduct_store_info gauge",
				fmt.Sprintf(`viaduct_store_info{backend=%q,persistent=%q} 1`, diagnostics.Backend, strconv.FormatBool(diagnostics.Persistent)),
			)
			if diagnostics.SchemaVersion > 0 {
				lines = append(lines,
					"# HELP viaduct_store_schema_version Latest applied store schema version.",
					"# TYPE viaduct_store_schema_version gauge",
					fmt.Sprintf("viaduct_store_schema_version %d", diagnostics.SchemaVersion),
				)
			}
			if diagnostics.DBPool != nil {
				lines = append(lines,
					"# HELP viaduct_store_db_pool_connections SQL connection pool connections by state.",
					"# TYPE viaduct_store_db_pool_connections gauge",
					fmt.Sprintf(`viaduct_store_db_pool_connections{state="max_open"} %d`, diagnostics.DBPool.MaxOpenConnections),
					fmt.Sprintf(`viaduct_store_db_pool_connections{state="open"} %d`, diagnostics.DBPool.OpenConnections),
					fmt.Sprintf(`viaduct_store_db_pool_connections{state="in_use"} %d`, diagnostics.DBPool.InUse),
					fmt.Sprintf(`viaduct_store_db_pool_connections{state="idle"} %d`, diagnostics.DBPool.Idle),
					"# HELP viaduct_store_db_pool_wait_total SQL connection pool wait events.",
					"# TYPE viaduct_store_db_pool_wait_total counter",
					fmt.Sprintf("viaduct_store_db_pool_wait_total %d", diagnostics.DBPool.WaitCount),
					"# HELP viaduct_store_db_pool_wait_seconds_total Total time spent waiting for a SQL connection.",
					"# TYPE viaduct_store_db_pool_wait_seconds_total counter",
					fmt.Sprintf("viaduct_store_db_pool_wait_seconds_total %.6f", diagnostics.DBPool.WaitDuration.Seconds()),
					"# HELP viaduct_store_db_pool_closures_total SQL connections closed by reason.",
					"# TYPE viaduct_store_db_pool_closures_total counter",
					fmt.Sprintf(`viaduct_store_db_pool_closures_total{reason="max_idle"} %d`, diagnostics.DBPool.MaxIdleClosed),
					fmt.Sprintf(`viaduct_store_db_pool_closures_total{reason="max_idle_time"} %d`, diagnostics.DBPool.MaxIdleTimeClosed),
					fmt.Sprintf(`viaduct_store_db_pool_closures_total{reason="max_lifetime"} %d`, diagnostics.DBPool.MaxLifetimeClosed),
					"# HELP viaduct_store_db_timeout_seconds Configured SQL query timeout by operation type.",
					"# TYPE viaduct_store_db_timeout_seconds gauge",
					fmt.Sprintf(`viaduct_store_db_timeout_seconds{type="read"} %.3f`, diagnostics.DBPool.ReadTimeout.Seconds()),
					fmt.Sprintf(`viaduct_store_db_timeout_seconds{type="write"} %.3f`, diagnostics.DBPool.WriteTimeout.Seconds()),
				)
			}
		}
	}

	tenants, err := s.store.ListTenants(ctx)
	if err != nil {
		packageLogger.Warn("failed to list tenants while rendering metrics", "error", err.Error())
		return strings.Join(lines, "\n") + "\n"
	}

	sort.Slice(tenants, func(i, j int) bool {
		return tenants[i].ID < tenants[j].ID
	})
	for _, tenant := range tenants {
		snapshots, err := s.store.ListSnapshots(store.ContextWithTenantID(ctx, tenant.ID), tenant.ID, "", 0)
		if err != nil {
			packageLogger.Warn("failed to list tenant snapshots while rendering metrics", "tenant_id", tenant.ID, "error", err.Error())
			continue
		}
		migrations, err := s.store.ListMigrations(store.ContextWithTenantID(ctx, tenant.ID), tenant.ID, 0)
		if err != nil {
			packageLogger.Warn("failed to list tenant migrations while rendering metrics", "tenant_id", tenant.ID, "error", err.Error())
			continue
		}

		lines = append(lines,
			fmt.Sprintf(`viaduct_tenant_snapshots{tenant=%q} %d`, tenant.ID, len(snapshots)),
			fmt.Sprintf(`viaduct_tenant_service_accounts{tenant=%q} %d`, tenant.ID, len(tenant.ServiceAccounts)),
		)

		phaseCounts := make(map[string]int)
		for _, migration := range migrations {
			phase := migration.Phase
			if phase == "" {
				phase = "unknown"
			}
			phaseCounts[phase]++
		}
		phases := make([]string, 0, len(phaseCounts))
		for phase := range phaseCounts {
			phases = append(phases, phase)
		}
		sort.Strings(phases)
		for _, phase := range phases {
			lines = append(lines, fmt.Sprintf(`viaduct_tenant_migrations{tenant=%q,phase=%q} %d`, tenant.ID, phase, phaseCounts[phase]))
		}

		if tenant.Quotas.MaxSnapshots > 0 {
			remaining := tenant.Quotas.MaxSnapshots - len(snapshots)
			if remaining < 0 {
				remaining = 0
			}
			lines = append(lines, fmt.Sprintf(`viaduct_tenant_quota_remaining{tenant=%q,resource="snapshots"} %d`, tenant.ID, remaining))
		}
		if tenant.Quotas.MaxMigrations > 0 {
			remaining := tenant.Quotas.MaxMigrations - len(migrations)
			if remaining < 0 {
				remaining = 0
			}
			lines = append(lines, fmt.Sprintf(`viaduct_tenant_quota_remaining{tenant=%q,resource="migrations"} %d`, tenant.ID, remaining))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}
