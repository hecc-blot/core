package errorcode

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 包级单例，模拟 leaf 模块 contract 里的 sentinel 定义方式。
var (
	errTimeout = New(CodeTimeout, "lock: acquire timeout")
	errBusy    = New(CodeServerBusy, "circuitbreaker: circuit open")
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "cache: key not found")
	assert.Equal(t, CodeNotFound, err.Code)
	assert.Equal(t, "cache: key not found", err.Error())
}

// TestFrom：直接命中、多级 %w 包裹、未携带码、nil 四种情形。
func TestFrom(t *testing.T) {
	code, ok := From(errTimeout)
	assert.True(t, ok)
	assert.Equal(t, CodeTimeout, code)

	wrapped := fmt.Errorf("handle order: %w", fmt.Errorf("acquire lock: %w", errTimeout))
	code, ok = From(wrapped)
	assert.True(t, ok)
	assert.Equal(t, CodeTimeout, code)

	_, ok = From(errors.New("plain error"))
	assert.False(t, ok)

	_, ok = From(nil)
	assert.False(t, ok)
}

// TestErrorsIsSentinel：单例语义 —— errors.Is 必须能穿过 %w 链路命中同一实例，
// 这是 A 类 sentinel（调用方据此决定重试/降级）成立的前提。
func TestErrorsIsSentinel(t *testing.T) {
	assert.True(t, errors.Is(errTimeout, errTimeout))
	assert.False(t, errors.Is(errTimeout, errBusy))

	wrapped := fmt.Errorf("acquire lock: %w", errTimeout)
	assert.True(t, errors.Is(wrapped, errTimeout))
	assert.False(t, errors.Is(wrapped, errBusy))
}

// TestNewReturnsDistinctInstance：New 每次返回新实例，故只能用于 sentinel 之外的临时错误；
// 用它构造 sentinel 会让 errors.Is 失效 —— 这条断言把这个约束固化成文档。
func TestNewReturnsDistinctInstance(t *testing.T) {
	a := New(CodeTimeout, "lock: acquire timeout")
	b := New(CodeTimeout, "lock: acquire timeout")
	assert.NotSame(t, a, b)
	assert.False(t, errors.Is(a, b))
}
