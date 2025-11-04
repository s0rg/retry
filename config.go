package retry

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"golang.org/x/sync/errgroup"
)

type mode byte

const (
	// Simple mode - time increases by sleep + jitter*attempt.
	Simple mode = 0
	// Linear mode - time increases by sleep*attempt + jitter.
	Linear mode = 1
	// Exponential mode - time increases by sleep*2^attempt + jitter.
	Exponential mode = 2
	// Fibonacci mode - time increases by sleep*fibonacci(attempt) + jitter.
	Fibonacci mode = 3
)

const (
	minParallel = 0
	minCount    = 1
	two         = 2
	minSleep    = time.Second / 2
	minDuration = time.Duration(0)
)

// Step represents a single execution step to re-try.
type Step struct {
	Func func() error
	Name string
}

// Config holds configuration.
type Config struct {
	fatal       []error
	sleep       time.Duration
	jitter      time.Duration
	count       int
	parallelism int
	mode        mode
	verbose     bool
}

// New creates new `Config` with given options
// If no options given default configuration will
// be applied: 1 retry in 1 second.
func New(opts ...option) (c *Config) {
	c = &Config{}

	for _, o := range opts {
		o(c)
	}

	c.validate()

	return c
}

// Single executes 'fn', until no error returned, at most `Count` times (default is 1,
// so `fn` will be executed at most 2 times), each execution delayed on time given
// as `Sleep` option (default is 1 second).
func (c *Config) Single(name string, fn func() error) (err error) {
	for n := range c.count {
		if err = fn(); err == nil {
			return nil
		}

		if c.isFatal(err) {
			break
		}

		if c.verbose {
			log.Printf("step %s:%d err: %v", name, n, err)
		}

		if n < c.count {
			time.Sleep(c.stepDuration(n + 1))
		}
	}

	return fmt.Errorf("%s: %w", name, err)
}

// Chain executes several `steps` one by one, returning first error.
func (c *Config) Chain(steps ...Step) (err error) {
	var step *Step

	for i := range len(steps) {
		step = &steps[i]

		if err = c.Single(step.Name, step.Func); err != nil {
			return fmt.Errorf("chain: %w", err)
		}
	}

	return nil
}

// Parallel executes several `steps` in parallel.
func (c *Config) Parallel(steps ...Step) (err error) {
	var eg errgroup.Group

	if c.parallelism > 0 {
		eg.SetLimit(c.parallelism)
	}

	for i := range len(steps) {
		step := steps[i]

		eg.Go(func() error {
			return c.Single(step.Name, step.Func)
		})
	}

	if err = eg.Wait(); err != nil {
		return fmt.Errorf("parallel: %w", err)
	}

	return nil
}

func (c *Config) validate() {
	if c.count < minCount {
		c.count = minCount
	}

	if c.sleep <= minDuration {
		c.sleep = minSleep
	}

	if c.jitter < minDuration {
		c.jitter = minDuration
	}

	if c.parallelism < minParallel {
		c.parallelism = minParallel
	}
}

func (c *Config) isFatal(err error) (yes bool) {
	for i := range len(c.fatal) {
		if errors.Is(c.fatal[i], err) {
			return true
		}
	}

	return false
}

func (c *Config) stepDuration(n int) (d time.Duration) {
	switch c.mode {
	case Linear:
		return c.sleep*time.Duration(n) + c.jitter
	case Exponential:
		return c.sleep*time.Duration(ipow2(n)) + c.jitter
	case Fibonacci:
		return c.sleep*time.Duration(fibonacci(n)) + c.jitter
	}

	return c.sleep + c.jitter*time.Duration(n)
}

func ipow2(v int) (rv int64) {
	return int64(math.Pow(two, float64(v)))
}

func fibonacci(n int) int64 {
	if n < two {
		return int64(n)
	}

	return fibonacci(n-1) + fibonacci(n-two)
}
