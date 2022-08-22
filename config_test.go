package retry

import (
	"errors"
	"testing"
	"time"
)

const maxTries = 3

var errFail = errors.New("test fail")

type failer struct {
	fun func()
	lim int
}

func newFailer(f func()) *failer {
	return &failer{fun: f}
}

func (f *failer) Fail() (err error) {
	f.fun()

	if f.lim > 0 { // start emitting errors when out of calls limit.
		f.lim--

		err = errFail
	}

	return
}

func (f *failer) Reset(limit int) {
	f.lim = limit
}

func TestSingle(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errCount    int
		countExpext int
		errExpect   error
	}{
		{1, 2, nil},
		{2, maxTries, nil},
		{maxTries, maxTries, errFail},
	}

	var (
		count int
		err   error
	)

	fail := newFailer(func() { count++ })

	try := New(
		Count(maxTries),
		Sleep(time.Millisecond),
	)

	for n, s := range table {
		fail.Reset(s.errCount)

		err = try.Single("test-case", fail.Fail)
		if !errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if count != s.countExpext {
			t.Fatalf("step %d: count = %d (want: %d)", n, count, s.countExpext)
		}

		count = 0
	}
}

func TestChain(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errCountA    int
		countAExpext int
		errCountB    int
		countBExpext int
		errExpect    error
	}{
		{1, 2, 0, 1, nil},
		{maxTries, maxTries, 0, 0, errFail},
		{1, 2, maxTries, maxTries, errFail},
	}

	var (
		countA int
		countB int
		err    error
	)

	fa := newFailer(func() { countA++ })
	fb := newFailer(func() { countB++ })

	try := New(
		Count(maxTries),
		Sleep(time.Millisecond),
		Verbose(true),
	)

	steps := []Step{
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

		if countA != s.countAExpext {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpext)
		}

		if countB != s.countBExpext {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpext)
		}

		countA, countB = 0, 0
	}
}

func TestParallel(t *testing.T) {
	t.Parallel()

	var table = []struct {
		errCountA    int
		countAExpext int
		errCountB    int
		countBExpext int
		errExpect    error
	}{
		{1, 2, 0, 1, nil},
		{maxTries, maxTries, 0, 1, errFail},
		{1, 2, maxTries, maxTries, errFail},
	}

	var (
		countA int
		countB int
		err    error
	)

	fa := newFailer(func() { countA++ })
	fb := newFailer(func() { countB++ })

	try := New(
		Count(maxTries),
		Sleep(time.Millisecond),
		Parallelism(2),
	)

	steps := []Step{
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

		if countA != s.countAExpext {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpext)
		}

		if countB < s.countBExpext {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpext)
		}

		countA, countB = 0, 0
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	try := New(
		Count(-10),
		Parallelism(-6),
		Sleep(-time.Hour),
		Jitter(-time.Minute),
		Verbose(true),
	)

	if try.count != minCount {
		t.Fatal("unexpected count")
	}

	if try.sleep != minSleep {
		t.Fatal("unexpected sleep")
	}

	if try.jitter != minDuration {
		t.Fatal("unexpected jitter")
	}

	if try.parallelism != minParallel {
		t.Fatal("unexpected parallelism")
	}

	if !try.verbose {
		t.Fatal("unexpected verbose")
	}
}
