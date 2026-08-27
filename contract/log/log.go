package log

import (
	"context"
)

// ILog 全仓通用日志契约：零第三方依赖，供所有模块（含 leaf 模块）复用。
// 实现方：framework/service/log（本地 zap）、log-sls（阿里云 SLS）等。
type ILog interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
}
