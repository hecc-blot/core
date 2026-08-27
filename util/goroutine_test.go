package util

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRecoverPanicCallsHandler(t *testing.T) {
	old := panicHandler
	defer func() { panicHandler = old }()

	var got any
	var gotStack []byte
	SetPanicHandler(func(r any, stack []byte) {
		got = r
		gotStack = stack
	})

	func() {
		defer recoverPanic()
		panic("boom")
	}()

	if got != "boom" {
		t.Fatalf("panic value = %v, want boom", got)
	}
	if len(gotStack) == 0 {
		t.Fatal("stack trace should not be empty")
	}
}

func TestGoRecoversPanic(t *testing.T) {
	old := panicHandler
	defer func() { panicHandler = old }()

	recovered := make(chan any, 1)
	SetPanicHandler(func(r any, _ []byte) {
		recovered <- r
	})

	Go(func() { panic("boom in goroutine") })

	got := <-recovered
	if got != "boom in goroutine" {
		t.Fatalf("recovered = %v, want boom in goroutine", got)
	}
}

func TestRunLimitCompletesAll(t *testing.T) {
	var count atomic.Int32
	tasks := make([]func(), 0, 20)
	for i := 0; i < 20; i++ {
		tasks = append(tasks, func() { count.Add(1) })
	}
	RunLimit(5, tasks)
	if got := count.Load(); got != 20 {
		t.Fatalf("completed %d tasks, want 20", got)
	}
}

func TestRunLimitBoundsConcurrency(t *testing.T) {
	var current, max atomic.Int32
	tasks := make([]func(), 0, 12)
	for i := 0; i < 12; i++ {
		tasks = append(tasks, func() {
			cur := current.Add(1)
			for {
				m := max.Load()
				if cur <= m || max.CompareAndSwap(m, cur) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			current.Add(-1)
		})
	}
	RunLimit(3, tasks)
	if got := max.Load(); got > 3 {
		t.Fatalf("max concurrency = %d, want <= 3", got)
	}
}
