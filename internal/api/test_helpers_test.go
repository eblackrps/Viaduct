package api

import (
	"testing"

	"github.com/eblackrps/viaduct/internal/store"
)

func mustNewServer(t *testing.T, stateStore store.Store) *Server {
	t.Helper()

	server, err := NewServer(nil, stateStore, 0, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}
