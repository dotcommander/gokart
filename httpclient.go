package gokart

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// HTTPConfig configures HTTP client behavior.
type HTTPConfig struct {
	Timeout   time.Duration // request timeout (default: 30s)
	RetryMax  int           // max retry attempts (default: 3)
	RetryWait time.Duration // wait between retries (default: 1s)
}

// NewHTTPClient creates a retryable HTTP client with exponential backoff.
//
// Default configuration:
//   - Timeout: 30s
//   - RetryMax: 3 attempts
//   - RetryWait: 1s base delay
//
// The client automatically retries on network errors and 5xx responses.
//
// Example:
//
//	client := gokart.NewHTTPClient(gokart.HTTPConfig{
//	    Timeout:   10 * time.Second,
//	    RetryMax:  5,
//	    RetryWait: 2 * time.Second,
//	})
//	resp, err := client.Get("https://api.example.com/data")
func NewHTTPClient(cfg HTTPConfig) *retryablehttp.Client {
	client := retryablehttp.NewClient()

	// Set timeout (default: 30s)
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client.HTTPClient.Timeout = timeout

	// Set max retries (default: 3)
	if cfg.RetryMax > 0 {
		client.RetryMax = cfg.RetryMax
	} else {
		client.RetryMax = 3
	}

	// Set retry wait time (default: 1s)
	if cfg.RetryWait > 0 {
		client.RetryWaitMin = cfg.RetryWait
		client.RetryWaitMax = cfg.RetryWait * 10 // max 10x the min wait
	} else {
		client.RetryWaitMin = 1 * time.Second
		client.RetryWaitMax = 10 * time.Second
	}

	return client
}

// NewStandardClient creates a standard http.Client with retry logic.
//
// This is a convenience wrapper around NewHTTPClient that returns a
// standard library http.Client interface for drop-in compatibility.
//
// Uses default configuration:
//   - Timeout: 30s
//   - RetryMax: 3 attempts
//   - RetryWait: 1s base delay
//
// Example:
//
//	client := gokart.NewStandardClient()
//	resp, err := client.Get("https://api.example.com/data")
func NewStandardClient() *http.Client {
	return NewHTTPClient(HTTPConfig{}).StandardClient()
}
