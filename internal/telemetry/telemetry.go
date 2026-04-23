package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeAPI identifies backend API spans.
	ScopeAPI = "github.com/eblackrps/viaduct/internal/api"
	// ScopeConnectors identifies connector client spans.
	ScopeConnectors = "github.com/eblackrps/viaduct/internal/connectors"
	// ScopeDiscovery identifies discovery-engine spans.
	ScopeDiscovery = "github.com/eblackrps/viaduct/internal/discovery"
	// ScopeMigrate identifies migration orchestrator spans.
	ScopeMigrate = "github.com/eblackrps/viaduct/internal/migrate"
	// ScopeStore identifies state-store spans.
	ScopeStore = "github.com/eblackrps/viaduct/internal/store"
)

const (
	defaultEnvironmentLocal  = "local"
	defaultEnvironmentRemote = "self-hosted"
	defaultOTLPEndpoint      = "http://127.0.0.1:4318"
	defaultSampler           = "parentbased_traceidratio"
	defaultServiceName       = "viaduct-api"
)

// Options configures Viaduct's OpenTelemetry setup.
type Options struct {
	Enabled        bool
	Endpoint       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	Sampler        string
	SamplerArg     float64
}

// OptionsFromEnv loads observability options from the current environment.
func OptionsFromEnv(serviceVersion string, localRuntime bool) Options {
	options := Options{
		Enabled:        strings.EqualFold(strings.TrimSpace(os.Getenv("VIADUCT_OTEL_ENABLED")), "true"),
		Endpoint:       strings.TrimSpace(os.Getenv("VIADUCT_OTEL_ENDPOINT")),
		ServiceName:    strings.TrimSpace(os.Getenv("VIADUCT_OTEL_SERVICE_NAME")),
		ServiceVersion: strings.TrimSpace(serviceVersion),
		Environment:    strings.TrimSpace(os.Getenv("VIADUCT_OTEL_ENVIRONMENT")),
		Sampler:        strings.TrimSpace(os.Getenv("VIADUCT_OTEL_SAMPLER")),
		SamplerArg:     1,
	}

	if options.Endpoint == "" {
		options.Endpoint = defaultOTLPEndpoint
	}
	if options.ServiceName == "" {
		options.ServiceName = defaultServiceName
	}
	if options.Environment == "" {
		if localRuntime {
			options.Environment = defaultEnvironmentLocal
		} else {
			options.Environment = defaultEnvironmentRemote
		}
	}
	if options.Sampler == "" {
		options.Sampler = defaultSampler
	}
	if raw := strings.TrimSpace(os.Getenv("VIADUCT_OTEL_SAMPLER_ARG")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			options.SamplerArg = parsed
		}
	}

	return options
}

// Setup configures the global OpenTelemetry tracer provider when enabled.
func Setup(ctx context.Context, options Options) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !options.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := newTraceExporter(ctx, options.Endpoint)
	if err != nil {
		return nil, err
	}

	resource, err := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewSchemaless(
			attribute.String("service.name", strings.TrimSpace(options.ServiceName)),
			attribute.String("service.version", strings.TrimSpace(options.ServiceVersion)),
			attribute.String("deployment.environment.name", strings.TrimSpace(options.Environment)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: build resource: %w", err)
	}

	sampler, err := newSampler(options.Sampler, options.SamplerArg)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

// StartSpan starts a named span from the requested scope.
func StartSpan(ctx context.Context, scope, name string, options ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(scope).Start(ctx, name, options...)
}

// CarrySpanContext copies the current span context from parent onto ctx.
func CarrySpanContext(ctx, parent context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if parent == nil {
		return ctx
	}
	spanContext := trace.SpanContextFromContext(parent)
	if !spanContext.IsValid() {
		return ctx
	}
	return trace.ContextWithSpanContext(ctx, spanContext)
}

// CurrentTraceID returns the current span trace ID string when present.
func CurrentTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}

// RecordError records an error on the provided span.
func RecordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// RecordHTTPStatus records an HTTP response status on the provided span.
func RecordHTTPStatus(span trace.Span, status int) {
	if span == nil || status <= 0 {
		return
	}
	span.SetAttributes(attribute.Int("http.response.status_code", status))
	if status >= http.StatusBadRequest {
		span.SetStatus(codes.Error, http.StatusText(status))
		return
	}
	span.SetStatus(codes.Ok, http.StatusText(status))
}

// HTTPClientTransport wraps an outbound HTTP transport with trace propagation and spans.
func HTTPClientTransport(base http.RoundTripper, component string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return tracedTransport{
		base:      base,
		component: strings.TrimSpace(component),
	}
}

type tracedTransport struct {
	base      http.RoundTripper
	component string
}

func (t tracedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return t.base.RoundTrip(req)
	}

	component := t.component
	if component == "" {
		component = "http.client"
	}

	ctx, span := StartSpan(req.Context(), ScopeConnectors, "connector.http", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		attribute.String("http.request.method", req.Method),
		attribute.String("url.path", firstNonEmpty(req.URL.EscapedPath(), "/")),
		attribute.String("server.address", req.URL.Hostname()),
		attribute.String("network.protocol.name", strings.ToLower(req.URL.Scheme)),
		attribute.String("viaduct.connector.component", component),
	))
	defer span.End()

	if port := req.URL.Port(); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			span.SetAttributes(attribute.Int("server.port", parsed))
		}
	}
	if sanitized := sanitizeURL(req.URL); sanitized != "" {
		span.SetAttributes(attribute.String("url.full", sanitized))
	}

	clone := req.Clone(ctx)
	clone.Header = clone.Header.Clone()
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(clone.Header))

	resp, err := t.base.RoundTrip(clone)
	if err != nil {
		RecordError(span, err)
		return nil, err
	}
	RecordHTTPStatus(span, resp.StatusCode)
	return resp, nil
}

func newTraceExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	options, err := otlpTraceHTTPOptions(endpoint)
	if err != nil {
		return nil, err
	}
	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("observability: create OTLP trace exporter: %w", err)
	}
	return exporter, nil
}

func otlpTraceHTTPOptions(endpoint string) ([]otlptracehttp.Option, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}

	options := []otlptracehttp.Option{otlptracehttp.WithTimeout(5 * time.Second)}
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("observability: parse OTLP endpoint: %w", err)
		}
		if strings.TrimSpace(parsed.Host) == "" {
			return nil, fmt.Errorf("observability: OTLP endpoint must include a host")
		}
		options = append(options, otlptracehttp.WithEndpoint(parsed.Host))
		if !strings.EqualFold(parsed.Scheme, "https") {
			options = append(options, otlptracehttp.WithInsecure())
		}
		path := strings.TrimSpace(parsed.Path)
		if path != "" && path != "/" {
			options = append(options, otlptracehttp.WithURLPath(path))
		}
		return options, nil
	}

	options = append(options, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	return options, nil
}

func newSampler(name string, arg float64) (sdktrace.Sampler, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", defaultSampler:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(clampSamplerArg(arg))), nil
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(clampSamplerArg(arg)), nil
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	default:
		return nil, fmt.Errorf("observability: unsupported sampler %q", name)
	}
}

func clampSamplerArg(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func sanitizeURL(target *url.URL) string {
	if target == nil {
		return ""
	}
	clone := *target
	clone.User = nil
	clone.RawQuery = ""
	clone.ForceQuery = false
	clone.Fragment = ""
	return clone.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
