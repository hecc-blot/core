package util

import (
	"context"

	"github.com/hecc-blot/core/enum/errorcode"
)

// ErrLimitFull 由 TryGo 返回：槽位已满，fn 未执行。对外码：errorcode.CodeRateLimited。
var ErrLimitFull = errorcode.New(errorcode.CodeRateLimited, "util: limit full")

// ErrLimitNotInitialized 由 Go / TryGo 返回：Limit 未用 NewLimit 构造。
// 注意 NewLimit 返回 *Limit，忘记构造拿到的是 nil 指针，此时若静默放行会退化成无界并发。
// 对外码：errorcode.CodeNotInitialized（编程错误，应当改代码而非重试）。
var ErrLimitNotInitialized = errorcode.New(errorcode.CodeNotInitialized, "util: limit not initialized, construct it with util.NewLimit")

// Limit 限制「同时执行的任务数」上限（本地信号量，非跨实例）。
//
// 构造一次、反复使用 —— 挂在拥有该受限资源的 service 上：
//
//	lim := util.NewLimit(64)              // 构造一次，生命周期同 service
//	lim.Go(ctx, func() { doWork(msg) })   // 每次调用共享同一计数器
//
// 不要每次调用前 NewLimit，那样每次都是新计数器，等于没限。
// 一个 Limit 对应一个受限资源（如 DB 一个、打下游 HTTP 一个），上限依据各自下游容量定。
//
// 边界：只约束经由它的调用点，管不住 fn 内部自起的协程，也管不住别处裸写的 go func(){}()。
// 它限的是「在途任务数」，不是「进程 goroutine 总数」。
//
// 用法选择：
//   - 流式（事件持续到来，如 MQ handler 内）→ Go（满则阻塞，形成背压）
//   - 不可阻塞、要明确拒绝（如 HTTP 入口）→ TryGo，满了返回 ErrLimitFull 由业务决定 429/丢弃
//   - 批式（给定一批任务、要全跑完）→ 用 RunLimit，不要用 Limit
type Limit struct {
	sem chan struct{}
}

// NewLimit 创建并发上限为 n 的限流器。n <= 0 归一为 1（与 RunLimit 一致）。
func NewLimit(n int) *Limit {
	if n <= 0 {
		n = 1
	}
	return &Limit{sem: make(chan struct{}, n)}
}

// Go 阻塞获取槽位，拿到后在协程内执行 fn（内部 recover panic，panic 日志带 traceId）。
// 槽位满则等待；等待期间 ctx 先结束 → 返回 ctx.Err()，fn 不执行。
//
// 返回非 nil 表示 fn 未启动，调用方必须处理：MQ 消费等场景若忽略并返回 nil，
// 会导致消息被 ack 但实际未处理。
//
// 不要在 fn 内部再对同一个 Limit 调 Go：槽位互相等待会死锁。
func (l *Limit) Go(ctx context.Context, fn func()) error {
	ctx = normalizeCtx(ctx)
	if l == nil || l.sem == nil {
		return ErrLimitNotInitialized
	}

	select {
	case l.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	// 复用 util.Go 的安全执行，panic 归因逻辑全仓只此一处。
	// 槽位在 panic 上报前先释放，故 panic handler 内再调本 Limit 也不会死锁。
	Go(ctx, func() {
		defer func() { <-l.sem }()
		fn()
	})
	return nil
}

// TryGo 不阻塞获取槽位：满则立即返回 ErrLimitFull；ctx 已结束则返回 ctx.Err()。
// 两者都表示 fn 未执行。
func (l *Limit) TryGo(ctx context.Context, fn func()) error {
	ctx = normalizeCtx(ctx)
	if l == nil || l.sem == nil {
		return ErrLimitNotInitialized
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case l.sem <- struct{}{}:
	default:
		return ErrLimitFull
	}

	Go(ctx, func() {
		defer func() { <-l.sem }()
		fn()
	})
	return nil
}

// InFlight 返回当前在途任务数。可周期采集进监控：长期贴着 Cap 说明该加下游容量或降上游速率。
func (l *Limit) InFlight() int {
	if l == nil || l.sem == nil {
		return 0
	}
	return len(l.sem)
}

// Cap 返回并发上限。
func (l *Limit) Cap() int {
	if l == nil || l.sem == nil {
		return 0
	}
	return cap(l.sem)
}
