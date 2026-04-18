package main

import "testing"

func TestNewServeAPICommand_DefaultsToLoopbackAndExposesDangerousOverride_Expected(t *testing.T) {
	t.Parallel()

	cmd := newServeAPICommand()

	hostFlag := cmd.Flags().Lookup("host")
	if hostFlag == nil {
		t.Fatal("host flag is missing")
	}
	if hostFlag.DefValue != "127.0.0.1" {
		t.Fatalf("host default = %q, want 127.0.0.1", hostFlag.DefValue)
	}

	overrideFlag := cmd.Flags().Lookup("allow-unauthenticated-remote")
	if overrideFlag == nil {
		t.Fatal("allow-unauthenticated-remote flag is missing")
	}
}
