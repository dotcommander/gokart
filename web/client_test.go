package web_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/dotcommander/gokart/web"
)

func TestTransportProductionTuning(t *testing.T) {
	t.Parallel()
	tr := web.Transport()
	if tr.MaxIdleConns != 100 {
		t.Fatalf("MaxIdleConns = %d, want 100", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost != 100 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 100 (default 2 kills throughput)", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 90*time.Second {
		t.Fatalf("IdleConnTimeout = %v, want 90s", tr.IdleConnTimeout)
	}
	// Clone must not mutate the global default.
	if got := http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost; got == 100 {
		t.Fatalf("Transport() mutated http.DefaultTransport (MaxIdleConnsPerHost = %d)", got)
	}
}

func TestNewClientUsesTunedTransport(t *testing.T) {
	t.Parallel()
	client := web.NewClient()
	if client == nil {
		t.Fatal("NewClient() = nil")
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.MaxIdleConnsPerHost != 100 {
		t.Fatalf("client transport MaxIdleConnsPerHost = %d, want 100", tr.MaxIdleConnsPerHost)
	}
}
