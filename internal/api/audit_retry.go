package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

const (
	defaultAuditRetryLogPath = "/var/lib/viaduct/audit-retry.log"
	auditRetryFlushInterval  = 5 * time.Second
)

type auditRetryEntry struct {
	Event models.AuditEvent `json:"event"`
}

func (s *Server) ensureAuditRetryFlusherStarted() {
	if s == nil {
		return
	}
	s.auditFlusherOnce.Do(func() {
		s.auditFlusherWG.Add(1)
		go func() {
			defer s.auditFlusherWG.Done()

			ticker := time.NewTicker(auditRetryFlushInterval)
			defer ticker.Stop()

			for {
				select {
				case <-s.backgroundTaskCtx.Done():
					if err := s.flushAuditRetryQueue(); err != nil && !errors.Is(err, os.ErrNotExist) {
						s.backgroundLogger().Warn("audit retry flusher failed during shutdown", "error", err.Error())
					}
					return
				case <-s.auditFlusherWake:
				case <-ticker.C:
				}

				if err := s.flushAuditRetryQueue(); err != nil && !errors.Is(err, os.ErrNotExist) {
					s.backgroundLogger().Warn("audit retry flusher failed", "error", err.Error())
				}
			}
		}()
	})
}

func (s *Server) signalAuditRetryFlusher() {
	if s == nil || s.auditFlusherWake == nil {
		return
	}
	select {
	case s.auditFlusherWake <- struct{}{}:
	default:
	}
}

func (s *Server) queueAuditRetryEvent(event models.AuditEvent) error {
	if s == nil {
		return fmt.Errorf("audit retry queue is not configured")
	}

	root, fileName, err := openAuditRetryRoot(s.auditRetryPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = root.Close()
	}()

	payload, err := json.Marshal(auditRetryEntry{Event: event})
	if err != nil {
		return fmt.Errorf("marshal audit retry event: %w", err)
	}

	s.auditRetryMu.Lock()
	defer s.auditRetryMu.Unlock()

	file, err := root.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open audit retry queue: %w", err)
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("append audit retry queue: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close audit retry queue: %w", err)
	}

	s.ensureAuditRetryFlusherStarted()
	s.signalAuditRetryFlusher()
	return nil
}

func openAuditRetryRoot(configuredPath string) (*os.Root, string, error) {
	path := strings.TrimSpace(configuredPath)
	if path == "" {
		path = defaultAuditRetryLogPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, "", fmt.Errorf("create audit retry directory: %w", err)
	}

	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, "", fmt.Errorf("open audit retry root: %w", err)
	}
	return root, filepath.Base(path), nil
}

func (s *Server) flushAuditRetryQueue() error {
	if s == nil || s.store == nil {
		return nil
	}

	root, fileName, err := openAuditRetryRoot(s.auditRetryPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = root.Close()
	}()

	s.auditRetryMu.Lock()
	defer s.auditRetryMu.Unlock()

	file, err := root.Open(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read audit retry queue: %w", err)
	}
	payload, err := io.ReadAll(file)
	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("close audit retry queue reader: %w", closeErr)
	}
	if err != nil {
		return fmt.Errorf("read audit retry queue payload: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(payload))
	failed := make([][]byte, 0)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var entry auditRetryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			s.backgroundLogger().Warn("dropping invalid audit retry entry", "error", err.Error())
			continue
		}

		ctx, cancel := s.auditRetryFlushContext(entry.Event.TenantID, entry.Event.RequestID)
		err := s.store.SaveAuditEvent(ctx, entry.Event)
		cancel()
		if err != nil {
			copied := append([]byte(nil), line...)
			failed = append(failed, copied)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan audit retry queue: %w", err)
	}

	if len(failed) == 0 {
		if err := root.Remove(fileName); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove audit retry queue: %w", err)
		}
		return nil
	}

	var buffer bytes.Buffer
	for _, line := range failed {
		buffer.Write(line)
		buffer.WriteByte('\n')
	}
	file, err = root.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open audit retry queue for rewrite: %w", err)
	}
	if _, err := file.Write(buffer.Bytes()); err != nil {
		_ = file.Close()
		return fmt.Errorf("rewrite audit retry queue payload: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync rewritten audit retry queue: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close rewritten audit retry queue: %w", err)
	}
	return nil
}

func (s *Server) auditRetryFlushContext(tenantID, requestID string) (context.Context, context.CancelFunc) {
	baseCtx := context.Background()
	if s != nil && s.backgroundTaskCtx != nil {
		baseCtx = context.WithoutCancel(s.backgroundTaskCtx)
	}

	ctx, cancel := context.WithCancel(baseCtx)
	if s != nil && s.backgroundTaskTimeout > 0 {
		cancel()
		ctx, cancel = context.WithTimeout(baseCtx, s.backgroundTaskTimeout)
	}

	ctx = store.ContextWithTenantID(ctx, tenantID)
	ctx = ContextWithRequestID(ctx, requestID)
	ctx = contextWithConnectorRequestID(ctx, requestID)
	return ctx, cancel
}
