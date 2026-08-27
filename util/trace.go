package util

import (
	"context"

	"github.com/hecc-blot/core/enum/trace"
)

// ExtractTraceId 从 context 提取链路追踪 ID（无则返回空串）。
// traceId 由 trace 模块写入 request.Context，键为自定义类型（trace.Value），
// 故直接 ctx.Value 取值即可，无需 gin 解包。
func ExtractTraceId(ctx context.Context) string {
	if v := ctx.Value(trace.TraceIdKey); v != nil {
		return v.(string)
	}
	return ""
}
