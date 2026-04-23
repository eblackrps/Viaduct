package store

import (
	"context"
	"strings"

	"github.com/eblackrps/viaduct/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func startPostgresStoreSpan(ctx context.Context, operation, tenantID string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tenantID = normalizeTenantID(tenantID)
	ctx = ContextWithTenantID(ctx, tenantID)

	baseAttrs := []attribute.KeyValue{
		attribute.String("viaduct.store.backend", "postgres"),
		attribute.String("viaduct.store.operation", strings.TrimSpace(operation)),
		attribute.String("tenant.id", tenantID),
	}
	baseAttrs = append(baseAttrs, attrs...)

	return telemetry.StartSpan(ctx, telemetry.ScopeStore, "store.postgres."+strings.TrimSpace(operation), trace.WithAttributes(baseAttrs...))
}

func finishStoreSpan(span trace.Span, err error, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	if err != nil {
		telemetry.RecordError(span, err)
	}
	span.End()
}

func appendStoreStringAttr(attrs []attribute.KeyValue, key, value string) []attribute.KeyValue {
	value = strings.TrimSpace(value)
	if value == "" {
		return attrs
	}
	return append(attrs, attribute.String(key, value))
}
