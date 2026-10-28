package daemon

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/config"
)

func newTestDaemon(t *testing.T) *GitDaemon {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.DataPath = t.TempDir()
	ctx := config.WithContext(context.Background(), cfg)
	d, err := NewGitDaemon(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestServeReturnsErrServerClosedOnShutdown asserts that Serve reports
// ErrServerClosed, not the raw accept error, when the daemon is shut down.
// Callers rely on this sentinel to distinguish a clean stop from a failure.
func TestServeReturnsErrServerClosedOnShutdown(t *testing.T) {
	for i := 0; i < 50; i++ {
		d := newTestDaemon(t)
		l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		errc := make(chan error, 1)
		go func() { errc <- d.Serve(l) }()

		// Wait until Serve has registered the listener.
		deadline := time.Now().Add(5 * time.Second)
		for {
			d.liMu.Lock()
			n := len(d.listeners)
			d.liMu.Unlock()
			if n > 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("Serve did not register its listener")
			}
			time.Sleep(time.Millisecond)
		}

		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
		select {
		case err := <-errc:
			if !errors.Is(err, ErrServerClosed) {
				t.Fatalf("iteration %d: expected ErrServerClosed, got %v", i, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Serve did not return after Shutdown")
		}
	}
}
