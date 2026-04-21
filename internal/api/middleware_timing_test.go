//go:build !race

package api

import (
	"context"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
)

var storedCredentialHashMatchesSink bool

func TestStoredCredentialHashMatches_TimingVarianceWithinThreshold_Expected(t *testing.T) {
	if testing.CoverMode() != "" {
		t.Skip("coverage instrumentation perturbs the micro-benchmark; release-gate runs this timing check without -cover")
	}

	validHash := hashCredential(context.Background(), "tenant-secret")
	legacyHash := store.HashAPIKey("tenant-secret")
	invalidHash := credentialHashPrefix + strings.Repeat("g", 64)
	cases := []struct {
		name string
		fn   func() bool
	}{
		{
			name: "zero",
			fn: func() bool {
				return storedCredentialHashMatches(context.Background(), legacyHash, "", [32]byte{})
			},
		},
		{
			name: "invalid",
			fn: func() bool {
				return storedCredentialHashMatches(context.Background(), invalidHash, "", validHash)
			},
		},
		{
			name: "wrong-length",
			fn: func() bool {
				return storedCredentialHashMatches(context.Background(), credentialHashPrefix+"abcd", "", validHash)
			},
		},
		{
			name: "correct",
			fn: func() bool {
				return storedCredentialHashMatches(context.Background(), legacyHash, "", validHash)
			},
		},
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)
	previousGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGCPercent)

	const (
		warmupRuns      = 10_000
		chunkIterations = 4_096
		chunksPerSample = 64
		samples         = 9
	)
	for _, tc := range cases {
		var matched bool
		for i := 0; i < warmupRuns; i++ {
			matched = tc.fn()
		}
		storedCredentialHashMatchesSink = matched
	}

	results := make(map[string][]float64, len(cases))
	const maxVariance = 0.20
	runtime.GC()
	for sample := 0; sample < samples; sample++ {
		sampleTotals := make(map[string]time.Duration, len(cases))
		for chunk := 0; chunk < chunksPerSample; chunk++ {
			for offset := 0; offset < len(cases); offset++ {
				tc := cases[(sample+chunk+offset)%len(cases)]
				startedAt := time.Now()
				var matched bool
				for i := 0; i < chunkIterations; i++ {
					matched = tc.fn()
				}
				storedCredentialHashMatchesSink = matched
				sampleTotals[tc.name] += time.Since(startedAt)
			}
		}
		totalIterations := float64(chunkIterations * chunksPerSample)
		for _, tc := range cases {
			results[tc.name] = append(results[tc.name], float64(sampleTotals[tc.name].Nanoseconds())/totalIterations)
		}
	}

	medians := make(map[string]float64, len(cases))
	for _, tc := range cases {
		sampleNs := results[tc.name]
		sort.Float64s(sampleNs)
		medians[tc.name] = sampleNs[len(sampleNs)/2]
	}

	minNs := medians["zero"]
	maxNs := medians["zero"]
	for _, ns := range medians {
		if ns < minNs {
			minNs = ns
		}
		if ns > maxNs {
			maxNs = ns
		}
	}

	if maxNs == 0 {
		t.Fatal("benchmark returned zero ns/op")
	}
	if variance := (maxNs - minNs) / maxNs; variance > maxVariance {
		t.Fatalf("storedCredentialHashMatches timing variance = %.3f, want <= %.3f; medians=%#v", variance, maxVariance, medians)
	}
}
