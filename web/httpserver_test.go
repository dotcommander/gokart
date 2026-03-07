package web

import (
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := DefaultServerConfig()
	cfg.ShutdownTimeout = 2 * time.Second

	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, "127.0.0.1:0", handler, cfg)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
