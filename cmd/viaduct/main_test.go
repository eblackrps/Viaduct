package main

import (
	"testing"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestExitCodeForError_ConfigurationConflictReturnsEXConfig_Expected(t *testing.T) {
	t.Parallel()

	if got := exitCodeForError(&store.CredentialConflictError{TenantID: "tenant-a"}); got != 78 {
		t.Fatalf("exitCodeForError(credential conflict) = %d, want 78", got)
	}
}

func TestExitCodeForError_DefaultsToGenericFailure_Expected(t *testing.T) {
	t.Parallel()

	if got := exitCodeForError(assertError("boom")); got != 1 {
		t.Fatalf("exitCodeForError(generic) = %d, want 1", got)
	}
}

type assertError string

func (e assertError) Error() string {
	return string(e)
}
