package util

import (
	"log"
	"runtime/debug"
	"sync"
)

// PanicHandler 处理协程内 panic，由上层注入（默认退回标准库 log）。
type PanicHandler func(r any, stack []byte)

var panicHandler PanicHandler = func(r any, stack []byte) {
	log.Printf("[goroutine panic] %v\n%s", r, stack)
}

// SetPanicHandler 注入 panic 处理（如接入业务 ILog）。非并发安全，应在启动时调用一次。
func SetPanicHandler(h PanicHandler) {
	if h != nil {
		panicHandler = h
	}
}

// recoverPanic 捕获并上报 panic，避免单个协程 panic 拖垮整个进程。
func recoverPanic() {
	if r := recover(); r != nil {
		panicHandler(r, debug.Stack())
	}
}

// Go 以 goroutine 安全执行 fn：非阻塞直起，内部 recover panic，不拖垮进程。
// 需要限制并发请用 RunLimit。
func Go(fn func()) {
	go func() {
		defer recoverPanic()
		fn()
	}()
}

// RunLimit 并发执行一批任务：最多 n 个同时运行、其余等待，等全部完成再返回（内部 recover panic）。
// 等价 errgroup.SetLimit 的有界扇出。
func RunLimit(n int, tasks []func()) {
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
				defer recoverPanic()
				t()
			}()
		}()
	}
	wg.Wait()
}
