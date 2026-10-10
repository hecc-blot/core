package util

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// PanicHandler 处理协程内 panic，由上层注入（默认退回标准库 log）。
// ctx 是起协程时传入的调用方上下文，供处理方取 traceId 归因。
type PanicHandler func(ctx context.Context, r any, stack []byte)

// defaultPanicHandler 未注入处理器时的兜底：标准库 log 输出，尽量带上 traceId。
var defaultPanicHandler PanicHandler = func(ctx context.Context, r any, stack []byte) {
	log.Printf("[goroutine panic] traceId=%q %v\n%s", ExtractTraceId(ctx), r, stack)
}

// panicHandler 当前生效的处理器，用 atomic 承载以允许启动之后安全替换。
var panicHandler atomic.Pointer[PanicHandler]

func init() {
	panicHandler.Store(&defaultPanicHandler)
}

// SetPanicHandler 注入 panic 处理（如接入业务 ILog）。传 nil 忽略、保持当前值。
// 并发安全，可在任意时刻调用；建议启动期设置一次。
func SetPanicHandler(h PanicHandler) {
	if h != nil {
		panicHandler.Store(&h)
	}
}

// recoverPanic 捕获并上报 panic，避免单个协程 panic 拖垮整个进程。
func recoverPanic(ctx context.Context) {
	if r := recover(); r != nil {
		if h := panicHandler.Load(); h != nil {
			(*h)(ctx, r, debug.Stack())
		}
	}
}

// normalizeCtx 保证 ctx 非 nil：nil 上取 Value/Done 会 panic。
func normalizeCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Go 以 goroutine 安全执行 fn：非阻塞直起，内部 recover panic，不拖垮进程。
// ctx 只用于 panic 归因（取 traceId），不控制 fn 的生命周期；需要取消或超时，
// 请在 fn 内自行监听 ctx.Done()。需要限制并发请用 RunLimit。
func Go(ctx context.Context, fn func()) {
	ctx = normalizeCtx(ctx)
	go func() {
		defer recoverPanic(ctx)
		fn()
	}()
}

// RunLimit 并发执行一批任务：最多 n 个同时运行、其余等待，等全部完成再返回（内部 recover panic）。
// 等价 errgroup.SetLimit 的有界扇出。ctx 只用于 panic 归因，不取消任务。
func RunLimit(ctx context.Context, n int, tasks []func()) {
	ctx = normalizeCtx(ctx)
	if n <= 0 {
		n = 1
	}
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		t := task
		sem <- struct{}{} // 满 n 时阻塞提交方，限制在途任务数 ≤ n
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			func() {
				defer recoverPanic(ctx)
				t()
			}()
		}()
	}
	wg.Wait()
}
