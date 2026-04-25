package main

import "testing"

func TestConfiguredStateStoreDSN_EnvOverridesConfig_Expected(t *testing.T) {
	t.Setenv("VIADUCT_STATE_STORE_DSN", "postgres://env-dsn")

	got := configuredStateStoreDSN(&appConfig{StateStoreDSN: "postgres://config-dsn"})
	if got != "postgres://env-dsn" {
		t.Fatalf("configuredStateStoreDSN() = %q, want env override", got)
	}
}

func TestConfiguredStateStoreDSN_ConfigFallback_Expected(t *testing.T) {
	got := configuredStateStoreDSN(&appConfig{StateStoreDSN: " postgres://config-dsn "})
	if got != "postgres://config-dsn" {
		t.Fatalf("configuredStateStoreDSN() = %q, want trimmed config DSN", got)
	}
}
