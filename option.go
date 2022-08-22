package retry

import "time"

type option func(*Config)

// Count sets number of retry attempts.
func Count(n int) func(*Config) {
	return func(s *Config) {
		s.count = n
	}
}

// Sleep sets sleep time between attempts.
func Sleep(d time.Duration) func(*Config) {
	return func(c *Config) {
		c.sleep = d
	}
}

// Jitter sets sleep-jitter value - if set, every attempt
// will await for sleep_value + jitter_value * attempt_number.
func Jitter(d time.Duration) func(*Config) {
	return func(c *Config) {
		c.jitter = d
	}
}

// Verbose sets verbosity of retry process.
func Verbose(v bool) func(*Config) {
	return func(c *Config) {
		c.verbose = v
	}
}

// Parallelism sets max parallelism count, zero (default) - indicates no limit.
func Parallelism(n int) func(*Config) {
	return func(c *Config) {
		c.parallelism = n
	}
}
