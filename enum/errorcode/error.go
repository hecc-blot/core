package errorcode

import "errors"

// Error 携带 Code 的 error。
//
// 用法：作为**包级单例**定义 sentinel，供调用方 errors.Is 判断后决定重试 / 降级：
//
//	// lock/contract
//	var ErrTimeout = errorcode.New(errorcode.CodeTimeout, "lock: acquire timeout")
//
// 之所以必须是单例：errors.Is 对同一实例走指针相等匹配，每次调用 New 得到的新实例
// 无法被 Is 命中。故本包只提供 New、**不提供 Newf** —— 需要格式化参数的属临时错误，
// 不是契约型信号，用 fmt.Errorf 配 %w 包一层即可（From 能穿透该链路取到码）。
type Error struct {
	Code Code // 错误码，渲染层与调用方据此处置
	msg  string
}

func (e *Error) Error() string {
	return e.msg
}

// New 构造携带码的 error，供包级单例使用（见 Error 的说明）。
func New(code Code, msg string) *Error {
	return &Error{Code: code, msg: msg}
}

// From 从错误链中提取码，支持 fmt.Errorf 的 %w 逐层包裹（Go 1.26 的 errors.AsType）。
// 未携带码（或 err 为 nil）时返回 false —— 调用方据此回退到默认处置。
func From(err error) (Code, bool) {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Code, true
	}
	return 0, false
}
