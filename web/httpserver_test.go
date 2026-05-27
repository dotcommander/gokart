package web

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultServerConfigNonZeroTimeouts(t *testing.T) {
	cfg := DefaultServerConfig()
	assert.Greater(t, cfg.ReadHeaderTimeout, time.Duration(0))
	assert.Greater(t, cfg.ReadTimeout, time.Duration(0))
	assert.Greater(t, cfg.WriteTimeout, time.Duration(0))
	assert.Greater(t, cfg.IdleTimeout, time.Duration(0))
	assert.Greater(t, cfg.MaxHeaderBytes, 0)
}

func TestServeContextCancellation(t *testing.T) {
	// Reserve a free port deterministically so we can poll it once Serve binds.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	addr := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatalf("release probe listener: %v", err)
	}

	// serveCtx drives Serve; cancelling it triggers graceful shutdown.
	// shutdownDeadlineCtx is the test's own upper bound for how long we wait for
	// Serve to return after cancel(). Splitting them avoids the select racing
	// between "Serve returned" and "the same ctx is Done".
	serveCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := DefaultServerConfig()
	cfg.ShutdownTimeout = 2 * time.Second

	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(serveCtx, addr, handler, cfg)
	}()

	// TCP-ready gate: assert.Eventually polls with the supplied tick interval
	// until the dial succeeds or the timeout expires.
	dialer := &net.Dialer{Timeout: 50 * time.Millisecond}
	ready := assert.Eventually(t, func() bool {
		conn, dialErr := dialer.DialContext(serveCtx, "tcp", addr)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 10*time.Millisecond, "server did not bind on %s", addr)
	if !ready {
		return
	}

	cancel()

	shutdownDeadlineCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-shutdownDeadlineCtx.Done():
		t.Fatal("server did not shut down before context deadline")
	}
}
