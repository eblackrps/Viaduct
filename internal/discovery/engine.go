package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Engine orchestrates discovery runs across one or more registered connectors.
type Engine struct {
	connectors map[string]connectors.Connector
	results    map[string]*models.DiscoveryResult
}

// PlatformSummary captures aggregate totals for a single source platform.
type PlatformSummary struct {
	// VMCount is the number of VMs discovered for the platform.
	VMCount int `json:"vm_count" yaml:"vm_count"`
	// TotalCPU is the aggregate vCPU count across all discovered VMs for the platform.
	TotalCPU int `json:"total_cpu" yaml:"total_cpu"`
	// TotalMemoryMB is the aggregate VM memory in mebibytes for the platform.
	TotalMemoryMB int64 `json:"total_memory_mb" yaml:"total_memory_mb"`
	// Source is the first source associated with the platform in the merged result.
	Source string `json:"source" yaml:"source"`
}

// MergedResult contains the aggregated output of one or more discovery runs.
type MergedResult struct {
	// Sources contains each source discovery result that completed successfully.
	Sources []models.DiscoveryResult `json:"sources" yaml:"sources"`
	// TotalVMs is the total number of discovered VMs across all sources.
	TotalVMs int `json:"total_vms" yaml:"total_vms"`
	// TotalCPU is the aggregate vCPU count across all discovered VMs.
	TotalCPU int `json:"total_cpu" yaml:"total_cpu"`
	// TotalMemoryMB is the aggregate VM memory in mebibytes across all sources.
	TotalMemoryMB int64 `json:"total_memory_mb" yaml:"total_memory_mb"`
	// ByPlatform captures per-platform summary totals.
	ByPlatform map[models.Platform]PlatformSummary `json:"by_platform" yaml:"by_platform"`
	// Errors contains non-fatal discovery or shutdown errors.
	Errors []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	// Duration is the total wall-clock time for the merged discovery run.
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// NewEngine creates a discovery engine with no registered sources.
func NewEngine() *Engine {
	return &Engine{
		connectors: make(map[string]connectors.Connector),
		results:    make(map[string]*models.DiscoveryResult),
	}
}

// AddSource registers a connector under the provided source name.
func (e *Engine) AddSource(name string, connector connectors.Connector) {
	if name == "" || connector == nil {
		return
	}

	e.connectors[name] = connector
}

// RunAll executes discovery for every registered connector concurrently and merges the results.
func (e *Engine) RunAll(ctx context.Context) (*MergedResult, error) {
	startedAt := time.Now()
	ctx, span := telemetry.StartSpan(ctx, telemetry.ScopeDiscovery, "discovery.run_all",
		trace.WithAttributes(attribute.Int("viaduct.discovery.source_count", len(e.connectors))),
	)
	defer span.End()
	e.results = make(map[string]*models.DiscoveryResult, len(e.connectors))

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		errs     []error
		messages []string
	)

	for name, connector := range e.connectors {
		name := name
		connector := connector

		wg.Add(1)
		go func() {
			defer wg.Done()
			sourceCtx, sourceSpan := telemetry.StartSpan(ctx, telemetry.ScopeDiscovery, "discovery.source",
				trace.WithAttributes(
					attribute.String("viaduct.discovery.source_name", name),
					attribute.String("viaduct.discovery.platform", string(connector.Platform())),
				),
			)
			defer sourceSpan.End()

			if err := runDiscoveryStep(sourceCtx, "discovery.connect", func(stepCtx context.Context) error {
				return connector.Connect(stepCtx)
			}); err != nil {
				telemetry.RecordError(sourceSpan, err)
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: connect: %w", name, err))
				messages = append(messages, fmt.Sprintf("%s: connect: %v", name, err))
				mu.Unlock()
				return
			}

			var result *models.DiscoveryResult
			err := runDiscoveryStep(sourceCtx, "discovery.discover", func(stepCtx context.Context) error {
				var discoverErr error
				result, discoverErr = connector.Discover(stepCtx)
				return discoverErr
			})
			if err != nil {
				telemetry.RecordError(sourceSpan, err)
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: discover: %w", name, err))
				messages = append(messages, fmt.Sprintf("%s: discover: %v", name, err))
				mu.Unlock()
			} else if result != nil {
				sourceSpan.SetAttributes(
					attribute.Int("viaduct.discovery.vm_count", len(result.VMs)),
					attribute.Int64("viaduct.discovery.duration_ms", result.Duration.Milliseconds()),
				)
				mu.Lock()
				e.results[name] = result
				mu.Unlock()
			}

			if closeErr := runDiscoveryStep(sourceCtx, "discovery.close", func(context.Context) error {
				return connector.Close()
			}); closeErr != nil {
				telemetry.RecordError(sourceSpan, closeErr)
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: close: %w", name, closeErr))
				messages = append(messages, fmt.Sprintf("%s: close: %v", name, closeErr))
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	merged := buildMergedResult(e.results, messages, time.Since(startedAt))
	if merged != nil {
		span.SetAttributes(
			attribute.Int("viaduct.discovery.result_count", len(merged.Sources)),
			attribute.Int("viaduct.discovery.total_vms", merged.TotalVMs),
			attribute.Int("viaduct.discovery.error_count", len(merged.Errors)),
		)
	}
	if len(errs) > 0 {
		telemetry.RecordError(span, errors.Join(errs...))
		return merged, errors.Join(errs...)
	}

	return merged, nil
}

func runDiscoveryStep(ctx context.Context, name string, run func(context.Context) error) error {
	stepCtx, span := telemetry.StartSpan(ctx, telemetry.ScopeDiscovery, name)
	defer span.End()
	if err := run(stepCtx); err != nil {
		telemetry.RecordError(span, err)
		return err
	}
	return nil
}

func buildMergedResult(results map[string]*models.DiscoveryResult, messages []string, duration time.Duration) *MergedResult {
	names := make([]string, 0, len(results))
	for name := range results {
		names = append(names, name)
	}
	sort.Strings(names)

	merged := &MergedResult{
		Sources:       make([]models.DiscoveryResult, 0, len(results)),
		ByPlatform:    make(map[models.Platform]PlatformSummary),
		Errors:        append([]string(nil), messages...),
		Duration:      duration,
		TotalMemoryMB: 0,
	}

	for _, name := range names {
		result := results[name]
		if result == nil {
			continue
		}

		merged.Sources = append(merged.Sources, *result)
		merged.TotalVMs += len(result.VMs)

		summary := merged.ByPlatform[result.Platform]
		if summary.Source == "" {
			summary.Source = result.Source
		}

		for _, vm := range result.VMs {
			merged.TotalCPU += vm.CPUCount
			merged.TotalMemoryMB += int64(vm.MemoryMB)
			summary.VMCount++
			summary.TotalCPU += vm.CPUCount
			summary.TotalMemoryMB += int64(vm.MemoryMB)
		}

		merged.ByPlatform[result.Platform] = summary
	}

	return merged
}
