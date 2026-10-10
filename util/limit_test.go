package util

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimitNewLimitNormalizes(t *testing.T) {
	for _, n := range []int{0, -1} {
		if got := NewLimit(n).Cap(); got != 1 {
			t.Fatalf("NewLimit(%d).Cap() = %d, want 1", n, got)
		}
	}
	if got := NewLimit(7).Cap(); got != 7 {
		t.Fatalf("NewLimit(7).Cap() = %d, want 7", got)
	}
}

// TestLimitBoundsConcurrency：并发扇出被限在 n 以内（提交方会被阻塞）。
func TestLimitBoundsConcurrency(t *testing.T) {
	const n, total = 3, 12
	lim := NewLimit(n)
	done := make(chan struct{}, total) // 必须缓冲，否则任务发不出 done 会占着槽位不放
	var current, max atomic.Int32

	for i := 0; i < total; i++ {
		if err := lim.Go(context.Background(), func() {
			cur := current.Add(1)
			for {
				m := max.Load()
				if cur <= m || max.CompareAndSwap(m, cur) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			current.Add(-1)
			done <- struct{}{}
		}); err != nil {
			t.Fatalf("Go: %v", err)
		}
	}
	for i := 0; i < total; i++ {
		<-done
	}

	if got := max.Load(); got > n {
		t.Fatalf("max concurrent = %d, want <= %d", got, n)
	}
	if got := lim.InFlight(); got != 0 {
		t.Fatalf("全部跑完后 InFlight = %d, want 0", got)
	}
}

// TestLimitGoBlocksWhenFull：满时阻塞提交方，释放后继续。
func TestLimitGoBlocksWhenFull(t *testing.T) {
	lim := NewLimit(1)
	started := make(chan struct{})
	release := make(chan struct{})

	if err := lim.Go(context.Background(), func() { close(started); <-release }); err != nil {
		t.Fatalf("第一个任务: %v", err)
	}
	<-started
	if got := lim.InFlight(); got != 1 {
		t.Fatalf("InFlight = %d, want 1", got)
	}

	second := make(chan struct{}, 1)
	go func() {
		if err := lim.Go(context.Background(), func() { second <- struct{}{} }); err != nil {
			t.Errorf("第二个任务: %v", err)
		}
	}()

	select {
	case <-second:
		t.Fatal("槽位未释放时第二个任务不应启动")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("槽位释放后第二个任务仍未启动")
	}
}

// TestLimitGoReturnsCtxErrWhenWaiting：等槽位期间 ctx 结束 → 报错且 fn 不执行。
func TestLimitGoReturnsCtxErrWhenWaiting(t *testing.T) {
	lim := NewLimit(1)
	started := make(chan struct{})
	release := make(chan struct{})

	if err := lim.Go(context.Background(), func() { close(started); <-release }); err != nil {
		t.Fatalf("占满槽位: %v", err)
	}
	<-started
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var ran atomic.Bool
	if err := lim.Go(ctx, func() { ran.Store(true) }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if ran.Load() {
		t.Fatal("ctx 已取消时 fn 不应执行")
	}
}

// TestLimitTryGoFullAndCancel：满 → ErrLimitFull；ctx 结束 → ctx.Err()。两者都不执行 fn。
func TestLimitTryGoFullAndCancel(t *testing.T) {
	lim := NewLimit(1)
	started := make(chan struct{})
	release := make(chan struct{})

	if err := lim.Go(context.Background(), func() { close(started); <-release }); err != nil {
		t.Fatalf("占满槽位: %v", err)
	}
	<-started
	defer close(release)

	var ran atomic.Bool
	if err := lim.TryGo(context.Background(), func() { ran.Store(true) }); !errors.Is(err, ErrLimitFull) {
		t.Fatalf("err = %v, want ErrLimitFull", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lim.TryGo(ctx, func() { ran.Store(true) }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	if ran.Load() {
		t.Fatal("满 / 取消时 fn 不应执行")
	}
}

func TestLimitTryGoSucceedsWhenSlotFree(t *testing.T) {
	lim := NewLimit(2)
	done := make(chan struct{}, 1)

	if err := lim.TryGo(context.Background(), func() { done <- struct{}{} }); err != nil {
		t.Fatalf("TryGo: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fn 未执行")
	}
}

// TestLimitRecoversPanicWithTraceId：Limit 复用 util.Go 的 panic 归因，且槽位在归因前已释放。
func TestLimitRecoversPanicWithTraceId(t *testing.T) {
	got := make(chan string, 1)
	swapPanicHandler(t, func(ctx context.Context, _ any, _ []byte) {
		got <- ExtractTraceId(ctx)
	})

	lim := NewLimit(1)
	if err := lim.Go(ctxWithTraceId("limit-trace"), func() { panic("boom") }); err != nil {
		t.Fatalf("Go: %v", err)
	}

	select {
	case id := <-got:
		if id != "limit-trace" {
			t.Fatalf("traceId = %q, want limit-trace", id)
		}
	case <-time.After(time.Second):
		t.Fatal("panic 未被处理器捕获")
	}

	// 槽位应在 panic 上报前释放（否则 panic handler 内再调本 Limit 会死锁）
	if got := lim.InFlight(); got != 0 {
		t.Fatalf("panic 后槽位未释放: InFlight = %d, want 0", got)
	}
}

// TestLimitNotInitialized：忘记 NewLimit 得到的是 nil 指针，必须报错而不是静默放行成无界并发。
func TestLimitNotInitialized(t *testing.T) {
	var nilLim *Limit
	if err := nilLim.Go(context.Background(), func() {}); !errors.Is(err, ErrLimitNotInitialized) {
		t.Fatalf("nil *Limit Go err = %v, want ErrLimitNotInitialized", err)
	}
	if err := nilLim.TryGo(context.Background(), func() {}); !errors.Is(err, ErrLimitNotInitialized) {
		t.Fatalf("nil *Limit TryGo err = %v, want ErrLimitNotInitialized", err)
	}
	if got := nilLim.InFlight(); got != 0 {
		t.Fatalf("nil InFlight = %d, want 0", got)
	}

	zero := &Limit{}
	if err := zero.Go(context.Background(), func() {}); !errors.Is(err, ErrLimitNotInitialized) {
		t.Fatalf("零值 Go err = %v, want ErrLimitNotInitialized", err)
	}
	if got := zero.Cap(); got != 0 {
		t.Fatalf("零值 Cap = %d, want 0", got)
	}
}

// TestLimitNilContextTolerated：nil ctx 归一为 Background，不应 panic。
func TestLimitNilContextTolerated(t *testing.T) {
	lim := NewLimit(1)
	ran := make(chan struct{}, 1)

	var nilCtx context.Context
	if err := lim.Go(nilCtx, func() { ran <- struct{}{} }); err != nil {
		t.Fatalf("Go: %v", err)
	}
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("fn 未执行")
	}
}
