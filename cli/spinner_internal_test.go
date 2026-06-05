package cli

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingWriter struct {
	mu     sync.Mutex
	writes int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	atomic.AddInt64(&w.writes, 1)
	return len(p), nil
}

func (w *countingWriter) count() int64 {
	return atomic.LoadInt64(&w.writes)
}

func TestSpinner_StopHaltsFramesPromptly(t *testing.T) {
	t.Parallel()

	w := &countingWriter{}
	s := NewSpinner("working").WithWriter(w).WithDelay(50 * time.Millisecond)
	s.forceTTY = true // drive the goroutine without a real TTY

	s.StartWithContext(context.Background())

	// Warm-up poll: wait until at least one frame is painted so the goroutine
	// is inside the inter-frame select. The sleep is a busy-wait between
	// checks, NOT a correctness assertion.
	deadline := time.Now().Add(2 * time.Second)
	for w.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if w.count() == 0 {
		t.Fatal("goroutine never painted a frame")
	}

	s.Stop()

	// Snapshot immediately after Stop, then wait longer than s.delay (50ms).
	// If the goroutine had a bare sleep, it would wake and paint another frame.
	after := w.count()
	time.Sleep(120 * time.Millisecond)
	final := w.count()

	if final != after {
		t.Errorf("frame written after Stop(): count went %d -> %d", after, final)
	}
}
