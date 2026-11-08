package gokart

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrTooLarge reports that input exceeded the caller-supplied limit.
var ErrTooLarge = errors.New("input exceeds limit")

// ReadAllLimit reads at most max bytes from r. If r holds more than max bytes
// it returns the max+1 bytes read together with an error wrapping ErrTooLarge.
//
// The infrastructure provides the bound; the application chooses the limit.
// The caller remains responsible for closing r.
func ReadAllLimit(r io.Reader, max int64) ([]byte, error) {
	if max < 0 {
		return nil, errors.New("max must be >= 0")
	}
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return data, fmt.Errorf("%w: more than %d bytes", ErrTooLarge, max)
	}
	return data, nil
}

// ReadFileLimit reads the file at path, returning an error wrapping
// ErrTooLarge if it exceeds max bytes.
func ReadFileLimit(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := ReadAllLimit(f, max)
	if err != nil {
		return data, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}
