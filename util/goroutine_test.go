package util

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hecc-blot/core/enum/trace"
)

// swapPanicHandler 临时替换处理器，测试结束恢复（含并行子测试）。
func swapPanicHandler(t *testing.T, h PanicHandler) {
	t.Helper()
	old := panicHandler.Load()
	SetPanicHandler(h)
	t.Cleanup(func() { panicHandler.Store(old) })
}

func ctxWithTraceId(id string) context.Context {
	return context.WithValue(context.Background(), trace.TraceIdKey, id)
}

func TestRecoverPanicCallsHandler(t *testing.T) {
	var got any
	var gotStack []byte
	swapPanicHandler(t, func(_ context.Context, r any, stack []byte) {
		got = r
		gotStack = stack
	})

	func() {
		defer recoverPanic(context.Background())
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
	recovered := make(chan any, 1)
	swapPanicHandler(t, func(_ context.Context, r any, _ []byte) {
		recovered <- r
	})

	Go(context.Background(), func() { panic("boom in goroutine") })

	if got := <-recovered; got != "boom in goroutine" {
		t.Fatalf("recovered = %v, want boom in goroutine", got)
	}
}

// TestGoPassesTraceId：起协程时把 ctx 透传给处理器，panic 日志才能归因到请求。
func TestGoPassesTraceId(t *testing.T) {
	got := make(chan string, 1)
	swapPanicHandler(t, func(ctx context.Context, _ any, _ []byte) {
		got <- ExtractTraceId(ctx)
	})

	Go(ctxWithTraceId("trace-abc"), func() { panic("boom") })

	if id := <-got; id != "trace-abc" {
		t.Fatalf("traceId = %q, want trace-abc", id)
	}
}

func TestRunLimitPassesTraceId(t *testing.T) {
	got := make(chan string, 1)
	swapPanicHandler(t, func(ctx context.Context, _ any, _ []byte) {
		got <- ExtractTraceId(ctx)
	})

	RunLimit(ctxWithTraceId("trace-xyz"), 1, []func(){func() { panic("boom") }})

	if id := <-got; id != "trace-xyz" {
		t.Fatalf("traceId = %q, want trace-xyz", id)
	}
}

// TestNilContextTolerated：ctx 为 nil 不应 panic（内部归一为 Background）。
// 用显式 nil 变量而非 nil 字面量，避免 SA1012（此处 nil 正是要验证的输入）。
func TestNilContextTolerated(t *testing.T) {
	got := make(chan string, 1)
	swapPanicHandler(t, func(ctx context.Context, _ any, _ []byte) {
		got <- ExtractTraceId(ctx)
	})

	var nilCtx context.Context
	Go(nilCtx, func() { panic("boom") })

	if id := <-got; id != "" {
		t.Fatalf("traceId = %q, want empty for nil ctx", id)
	}
}

// TestSetPanicHandlerNilKeepsCurrent：传 nil 是 no-op，不应把处理器置空。
func TestSetPanicHandlerNilKeepsCurrent(t *testing.T) {
	called := make(chan struct{}, 1)
	swapPanicHandler(t, func(_ context.Context, _ any, _ []byte) {
		called <- struct{}{}
	})

	SetPanicHandler(nil)

	func() {
		defer recoverPanic(context.Background())
		panic("boom")
	}()

	select {
	case <-called:
	default:
		t.Fatal("传 nil 后处理器被清空，panic 未被上报")
	}
}

func TestRunLimitCompletesAll(t *testing.T) {
	var count atomic.Int32
	tasks := make([]func(), 0, 20)
	for i := 0; i < 20; i++ {
		tasks = append(tasks, func() { count.Add(1) })
	}
	RunLimit(context.Background(), 5, tasks)
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
	RunLimit(context.Background(), 3, tasks)
	if got := max.Load(); got > 3 {
		t.Fatalf("max concurrency = %d, want <= 3", got)
	}
}
