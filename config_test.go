package retry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/s0rg/retry"
)

const maxTries = 3

var (
	errFail  = errors.New("test fail")
	errFatal = errors.New("custom fatal error")
)

type failer struct {
	err error
	fun func()
	lim int
}

func newFailer(err error, f func()) *failer {
	return &failer{fun: f, err: err}
}

func (f *failer) Fail() (err error) {
	f.fun()

	if f.lim > 0 { // start emitting errors when out of calls limit.
		f.lim--

		err = f.err
	}

	return
}

func (f *failer) Reset(limit int) {
	f.lim = limit
}

func TestSingle(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errExpect   error
		errCount    int
		countExpect int
	}{
		{
			errCount:    1,
			countExpect: 2,
			errExpect:   nil,
		},
		{
			errCount:    2,
			countExpect: maxTries,
			errExpect:   nil,
		},
		{
			errCount:    maxTries,
			countExpect: maxTries,
			errExpect:   errFail,
		},
	}

	var (
		count int
		err   error
	)

	fail := newFailer(errFail, func() { count++ })

	try := retry.New(
		retry.Count(maxTries),
		retry.Sleep(time.Millisecond),
		retry.Mode(retry.Linear),
	)

	for n, s := range table {
		fail.Reset(s.errCount)

		err = try.Single("test-case", fail.Fail)
		if !errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if count != s.countExpect {
			t.Fatalf("step %d: count = %d (want: %d)", n, count, s.countExpect)
		}

		count = 0
	}
}

func TestChain(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errExpect    error
		errCountA    int
		countAExpect int
		errCountB    int
		countBExpect int
	}{
		{
			errCountA:    1,
			countAExpect: 2,
			errCountB:    0,
			countBExpect: 1,
			errExpect:    nil,
		},
		{
			errCountA:    maxTries,
			countAExpect: maxTries,
			errCountB:    0,
			countBExpect: 0,
			errExpect:    errFail,
		},
		{
			errCountA:    1,
			countAExpect: 2,
			errCountB:    maxTries,
			countBExpect: maxTries,
			errExpect:    errFail,
		},
	}

	var (
		countA int
		countB int
		err    error
	)

	fa := newFailer(errFail, func() { countA++ })
	fb := newFailer(errFail, func() { countB++ })

	try := retry.New(
		retry.Count(maxTries),
		retry.Sleep(time.Millisecond),
		retry.Verbose(true),
		retry.Mode(retry.Exponential),
	)

	steps := []retry.Step{
		{Name: "chain-A", Func: fa.Fail},
		{Name: "chain-B", Func: fb.Fail},
	}

	for n, s := range table {
		fa.Reset(s.errCountA)
		fb.Reset(s.errCountB)

		err = try.Chain(steps...)
		if !errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if countA != s.countAExpect {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpect)
		}

		if countB != s.countBExpect {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpect)
		}

		countA, countB = 0, 0
	}
}

func TestParallel(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errExpect    error
		errCountA    int
		countAExpect int
		errCountB    int
		countBExpect int
	}{
		{
			errCountA:    1,
			countAExpect: 2,
			errCountB:    0,
			countBExpect: 1,
			errExpect:    nil,
		},
		{
			errCountA:    maxTries,
			countAExpect: maxTries,
			errCountB:    0,
			countBExpect: 1,
			errExpect:    errFail,
		},
		{
			errCountA:    1,
			countAExpect: 2,
			errCountB:    maxTries,
			countBExpect: maxTries,
			errExpect:    errFail,
		},
	}

	var (
		countA int
		countB int
		err    error
	)

	fa := newFailer(errFail, func() { countA++ })
	fb := newFailer(errFail, func() { countB++ })

	try := retry.New(
		retry.Count(maxTries),
		retry.Sleep(time.Millisecond),
		retry.Jitter(time.Millisecond),
		retry.Parallelism(2),
	)

	steps := []retry.Step{
		{Name: "parallel-A", Func: fa.Fail},
		{Name: "parallel-B", Func: fb.Fail},
	}

	for n, s := range table {
		fa.Reset(s.errCountA)
		fb.Reset(s.errCountB)

		err = try.Parallel(steps...)
		if !errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if countA != s.countAExpect {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpect)
		}

		if countB < s.countBExpect {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpect)
		}

		countA, countB = 0, 0
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	try := retry.New(
		retry.Count(-10),
		retry.Parallelism(-6),
		retry.Sleep(-time.Hour),
		retry.Jitter(-time.Minute),
	)

	c := 0

	f := func() error {
		c++

		return nil
	}

	if err := try.Single("test-validate", f); err != nil {
		t.Error("test-validate", err)
	}

	if c < 1 {
		t.Error("unexpected count")
	}
}

func TestFibonacci(t *testing.T) {
	t.Parallel()

	var counter int

	const (
		tryMax   = 5
		errCount = 4
	)

	try := retry.New(
		retry.Count(tryMax),
		retry.Mode(retry.Fibonacci),
	)

	err := try.Single("test-fibonacci", func() (err error) {
		counter++

		if counter < errCount {
			return errFail
		}

		return nil
	})
	if err != nil {
		t.FailNow()
	}

	if counter != errCount {
		t.FailNow()
	}
}

func TestFatal(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errExpect    error
		errCountA    int
		countAExpect int
		errCountB    int
		countBExpect int
	}{
		{
			errCountA:    1,
			countAExpect: 1,
			errCountB:    0,
			countBExpect: 0,
			errExpect:    errFatal,
		},
	}

	var (
		countA int
		countB int
		err    error
	)

	fa := newFailer(errFatal, func() { countA++ })
	fb := newFailer(errFail, func() { countB++ })

	try := retry.New(
		retry.Count(maxTries),
		retry.Fatal(errFatal),
		retry.Mode(retry.Fibonacci),
	)

	steps := []retry.Step{
		{Name: "parallel-A", Func: fa.Fail},
		{Name: "parallel-B", Func: fb.Fail},
	}

	for n, s := range table {
		fa.Reset(s.errCountA)
		fb.Reset(s.errCountB)

		err = try.Chain(steps...)
		if !errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if countA != s.countAExpect {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpect)
		}

		if countB < s.countBExpect {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpect)
		}

		countA, countB = 0, 0
	}
}
