package api

import (
	"net/http/httptest"
	"testing"
)

func TestParseConcreteIP_RejectsZoneQualifiedInputs_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "plain ipv4", value: "127.0.0.1", want: "127.0.0.1", ok: true},
		{name: "plain ipv6", value: "::1", want: "::1", ok: true},
		{name: "ipv6 zone rejected", value: "fe80::1%eth0", ok: false},
		{name: "ipv6 loopback zone rejected", value: "::1%lo", ok: false},
		{name: "ipv4 zone rejected", value: "127.0.0.1%0", ok: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseConcreteIP(tc.value)
			if ok != tc.ok {
				t.Fatalf("parseConcreteIP(%q) ok = %t, want %t", tc.value, ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if got.String() != tc.want {
				t.Fatalf("parseConcreteIP(%q) = %q, want %q", tc.value, got.String(), tc.want)
			}
		})
	}
}

func TestForwardedForIP_RejectsNonCanonicalEntries_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "canonical ipv4", value: "127.0.0.1", want: "127.0.0.1", ok: true},
		{name: "canonical ipv6", value: "::1", want: "::1", ok: true},
		{name: "ipv6 zone rejected", value: "::1%lo", ok: false},
		{name: "link local zone rejected", value: "fe80::1%eth0", ok: false},
		{name: "ipv4 zone rejected", value: "127.0.0.1%0", ok: false},
		{name: "ipv4 mapped rejected", value: "::ffff:127.0.0.1", ok: false},
		{name: "ipv4 mapped remote rejected", value: "::ffff:8.8.8.8", ok: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := forwardedForIP(tc.value)
			if ok != tc.ok {
				t.Fatalf("forwardedForIP(%q) ok = %t, want %t", tc.value, ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if got.String() != tc.want {
				t.Fatalf("forwardedForIP(%q) = %q, want %q", tc.value, got.String(), tc.want)
			}
		})
	}
}

func TestRejectedForwardingHeader_PrefersInvalidForwardedForOverRealIP_Expected(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/about", nil)
	req.Header.Set("X-Forwarded-For", "::1%lo")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	if got := rejectedForwardingHeader(req); got != "X-Forwarded-For" {
		t.Fatalf("rejectedForwardingHeader() = %q, want X-Forwarded-For", got)
	}
}

func TestRejectedForwardingHeaders_ReportsEveryInvalidHeader_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		forwarded string
		realIP    string
		want      []string
	}{
		{name: "xff only", forwarded: "::1%lo", want: []string{"X-Forwarded-For"}},
		{name: "x-real-ip only", realIP: "127.0.0.1%0", want: []string{"X-Real-IP"}},
		{name: "both", forwarded: "::1%lo", realIP: "127.0.0.1%0", want: []string{"X-Forwarded-For", "X-Real-IP"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/about", nil)
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			if tc.realIP != "" {
				req.Header.Set("X-Real-IP", tc.realIP)
			}

			got := rejectedForwardingHeaders(req)
			if len(got) != len(tc.want) {
				t.Fatalf("rejectedForwardingHeaders() = %#v, want %#v", got, tc.want)
			}
			for index := range tc.want {
				if got[index] != tc.want[index] {
					t.Fatalf("rejectedForwardingHeaders() = %#v, want %#v", got, tc.want)
				}
			}
		})
	}
}
