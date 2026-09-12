package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type iInterface interface {
	Test() string
}

type derive struct{}

func (m derive) Test() string {
	return "set test"
}

type defaultTest struct {
	One iInterface `inject:""`
}

type customTest struct {
	One iInterface `inject:"custom"`
}

type composeTest struct {
	defaultTest

	Child iInterface `inject:"custom"`
}

func TestContainer(t *testing.T) {
	container := New()
	container.Set(new(iInterface), derive{})
	container.SetWithName(new(iInterface), "custom", derive{})

	t.Run("default", func(t *testing.T) {
		var d1 defaultTest
		container.Inject(&d1)

		assert.Equal(t, "set test", d1.One.Test())
	})

	t.Run("custom", func(t *testing.T) {
		var d2 customTest
		container.Inject(&d2)

		assert.Equal(t, "set test", d2.One.Test())
	})

	t.Run("compose", func(t *testing.T) {
		d3 := composeTest{}
		container.Inject(&d3)

		assert.Equal(t, "set test", d3.One.Test())
		assert.Equal(t, "set test", d3.defaultTest.One.Test())
		assert.Equal(t, "set test", d3.Child.Test())
	})
}

func TestContainerEdgeCases(t *testing.T) {
	t.Run("Inject 非指针 panic", func(t *testing.T) {
		container := New()
		var d defaultTest
		assert.Panics(t, func() {
			container.Inject(d)
		})
	})

	t.Run("Set 未实现接口 panic", func(t *testing.T) {
		container := New()
		assert.Panics(t, func() {
			container.Set(new(iInterface), "not implementing")
		})
	})

	t.Run("Get 未注册 panic", func(t *testing.T) {
		container := New()
		assert.Panics(t, func() {
			container.Get(new(iInterface), "")
		})
	})

	t.Run("多容器隔离", func(t *testing.T) {
		c1 := New()
		c2 := New()
		c1.Set(new(iInterface), derive{})

		var d defaultTest
		assert.Panics(t, func() {
			c2.Inject(&d)
		})

		c1.Inject(&d)
		assert.Equal(t, "set test", d.One.Test())
	})
}
