package errorcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsCoreSegment：段位是 framework 渲染层区分「基础设施码 / 协议码」的唯一依据，
// 边界必须精确（含端点），相邻段不得误判。
func TestIsCoreSegment(t *testing.T) {
	cases := []struct {
		code Code
		want bool
	}{
		{0, false},
		{9999, false},
		{10000, false}, // framework：Success
		{10001, false}, // framework：Processing
		{19999, false},
		{20000, true}, // 段下界（含）
		{25000, true},
		{29999, true}, // 段上界（含）
		{30000, false},
		{40001, false}, // framework：Busy
		{60001, false}, // 业务自定义
	}
	for _, c := range cases {
		assert.Equal(t, c.want, IsCoreSegment(c.code), "code=%d", c.code)
	}
}

// TestCoreCodesWellFormed：core 段新增码时，同时守住三条不变量 ——
// 落在段内、互不重复、有对外文案（透传路径靠它出 message）。
func TestCoreCodesWellFormed(t *testing.T) {
	codes := []Code{
		CodeServerBusy,
		CodeTimeout,
		CodeRateLimited,
		CodeProcessing,
		CodeConflict,
		CodeIllegalState,
		CodeNotInitialized,
		CodeNotFound,
	}

	seen := make(map[Code]bool, len(codes))
	for _, code := range codes {
		assert.True(t, IsCoreSegment(code), "码 %d 不在 core 段内", code)
		assert.False(t, seen[code], "码 %d 重复定义", code)
		seen[code] = true
		assert.NotEmpty(t, Message(code), "码 %d 缺少文案", code)
	}
}

// TestMessageUnknownCode：非 core 段的码没有通用文案，返回空串让调用方回退到 err.Error()。
func TestMessageUnknownCode(t *testing.T) {
	assert.Empty(t, Message(0))
	assert.Empty(t, Message(40002)) // framework：ValidateError
	assert.Empty(t, Message(60001)) // 业务自定义
}
