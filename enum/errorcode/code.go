// Package errorcode 提供全仓通用的错误码，以及「携带错误码的 error」。
//
// 定位：错误码的**定义处**。core 定义基础设施通用段，framework 决定对外暴露哪些
// （渲染层可逐码翻译，见 framework/enum/response），业务在 60000+ 段自行扩展。
// 契约层保持零第三方依赖（纯 stdlib），任何 leaf 模块都可安全引用，
// 因此 leaf 模块的 sentinel 只能带本包的码 —— 不能反向引用 framework 的协议码。
package errorcode

// Code 通用错误码。
//
// 段位划分（全仓约定；**已发布的码不可复用或改语义**）：
//
//	0 – 9999        留白（预留给成功 / 处理中的细分类）
//	10000 – 10001   framework 协议：成功 / 处理中（已发布，不可动）
//	20000 – 29999   core 通用（本包，基础设施语义）
//	40000 – 59999   framework 协议（对客户端可见）
//	60000 +         业务自定义（业务内再按业务域分段，如 60xxx 订单、61xxx 支付）
type Code int

// core 通用段（20000 – 29999）。
//
// 每个码对应一类「基础设施语义」，而不是某次具体的失败原因 —— 调用方据此决定
// 重试 / 降级 / 直接失败，所以划分粒度到「处置方式」为止，不再往下细分。
const (
	CodeServerBusy     Code = 20000 // 服务器繁忙（下游熔断 / 不可用）
	CodeTimeout        Code = 20001 // 等待超时
	CodeRateLimited    Code = 20002 // 超出限流 / 并发上限
	CodeProcessing     Code = 20003 // 处理中（重复请求正在处理）
	CodeConflict       Code = 20004 // 资源被占用（并发冲突，如锁已被他人持有）
	CodeIllegalState   Code = 20005 // 状态非法（前置条件不满足，如未持有锁）
	CodeNotInitialized Code = 20006 // 组件未初始化（编程错误，应当改代码而非重试）
	CodeNotFound       Code = 20007 // 资源不存在
)

// core 通用段边界（含端点）。
const (
	coreSegmentMin Code = 20000
	coreSegmentMax Code = 29999
)

// IsCoreSegment 判断码是否落在 core 通用段内。
//
// framework 渲染层用它区分「需要按映射表翻译的基础设施码」与「协议码 / 业务码」：
// 前者未配置映射时可直接透传（并记一条日志），后者透传即是对外契约本身。
func IsCoreSegment(code Code) bool {
	return code >= coreSegmentMin && code <= coreSegmentMax
}

// messageText core 段文案表。
//
// 用途：framework 渲染层对客户端输出 message 时，按
// 「协议码文案 → core 段文案 → err.Error()」的顺序回退；
// 没有这张表，透传路径的 core 码在客户端会拿到空 message。
var messageText = map[Code]string{
	CodeServerBusy:     "server busy",
	CodeTimeout:        "request timeout",
	CodeRateLimited:    "too many requests",
	CodeProcessing:     "processing",
	CodeConflict:       "resource conflict",
	CodeIllegalState:   "illegal state",
	CodeNotInitialized: "service not ready", // 编程错误，对外不明示细节
	CodeNotFound:       "not found",
}

// Message 返回码的通用文案，未知码返回空串（调用方自行回退到 err.Error()）。
func Message(code Code) string {
	return messageText[code]
}
