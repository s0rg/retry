package retry

// Step represents a single execution step to re-try.
type Step struct {
	Name string
	Func func() error
}
