package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Spinner shows an animated spinner with a message.
type Spinner struct {
	message string
	frames  []string
	delay   time.Duration
	writer  io.Writer
	done    chan struct{}
	mu      sync.Mutex
	running bool
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner with a message.
//
// Example:
//
//	s := cli.NewSpinner("Loading...")
//	s.Start()
//	// do work
//	s.Stop()
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  defaultFrames,
		delay:   80 * time.Millisecond,
		writer:  os.Stdout,
		done:    make(chan struct{}),
	}
}

// WithFrames sets custom animation frames.
func (s *Spinner) WithFrames(frames []string) *Spinner {
	s.frames = frames
	return s
}

// WithDelay sets the animation speed.
func (s *Spinner) WithDelay(d time.Duration) *Spinner {
	s.delay = d
	return s
}

// WithWriter sets the output writer.
func (s *Spinner) WithWriter(w io.Writer) *Spinner {
	s.writer = w
	return s
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()

				fmt.Fprintf(s.writer, "\r%s %s", s.frames[i%len(s.frames)], msg)
				i++
				time.Sleep(s.delay)
			}
		}
	}()
}

// Update changes the spinner message.
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.done)
	s.mu.Unlock()

	// Clear the line
	fmt.Fprintf(s.writer, "\r\033[K")
}

// StopWithMessage stops and prints a final message.
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, message)
}

// StopSuccess stops and prints a success message.
func (s *Spinner) StopSuccess(message string) {
	s.Stop()
	Success("%s", message)
}

// StopError stops and prints an error message.
func (s *Spinner) StopError(message string) {
	s.Stop()
	Error("%s", message)
}

// WithSpinner runs a function with a spinner, handling success/error.
//
// Example:
//
//	err := cli.WithSpinner("Processing...", func() error {
//	    return doSomething()
//	})
func WithSpinner(message string, fn func() error) error {
	s := NewSpinner(message)
	s.Start()

	err := fn()

	if err != nil {
		s.StopError(fmt.Sprintf("%s failed: %v", message, err))
	} else {
		s.StopSuccess(message + " done")
	}

	return err
}

// Progress shows a simple progress indicator.
type Progress struct {
	total   int
	current int
	message string
	writer  io.Writer
	mu      sync.Mutex
}

// NewProgress creates a progress indicator.
//
// Example:
//
//	p := cli.NewProgress("Processing files", 100)
//	for i := 0; i < 100; i++ {
//	    p.Increment()
//	    // do work
//	}
//	p.Done()
func NewProgress(message string, total int) *Progress {
	return &Progress{
		total:   total,
		message: message,
		writer:  os.Stdout,
	}
}

// SetWriter sets the output writer.
func (p *Progress) SetWriter(w io.Writer) *Progress {
	p.writer = w
	return p
}

// Increment advances the progress by 1.
func (p *Progress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++
	p.render()
}

// Set sets the current progress value.
func (p *Progress) Set(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	p.render()
}

// render draws the progress bar.
func (p *Progress) render() {
	pct := float64(p.current) / float64(p.total) * 100
	barWidth := 30
	filled := int(float64(barWidth) * float64(p.current) / float64(p.total))

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	fmt.Fprintf(p.writer, "\r%s [%s] %d/%d (%.0f%%)", p.message, bar, p.current, p.total, pct)
}

// Done completes the progress and moves to a new line.
func (p *Progress) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
	fmt.Fprintln(p.writer)
}
