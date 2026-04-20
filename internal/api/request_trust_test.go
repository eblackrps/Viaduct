package api

import "testing"

func TestParseConcreteIP_DropsIPv6ZonesAndRejectsInvalidZoneForms_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "plain ipv4", value: "127.0.0.1", want: "127.0.0.1", ok: true},
		{name: "plain ipv6", value: "::1", want: "::1", ok: true},
		{name: "ipv6 zone dropped", value: "fe80::1%eth0", want: "fe80::1", ok: true},
		{name: "ipv6 loopback zone dropped", value: "::1%lo", want: "::1", ok: true},
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
