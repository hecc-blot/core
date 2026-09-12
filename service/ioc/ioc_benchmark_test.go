package service

import "testing"

// BenchmarkInject 度量反射注入一个含匿名嵌套与命名注入的结构体开销。
// 复用 ioc_svc_test.go 中定义的 iInterface/derive/composeTest。
func BenchmarkInject(b *testing.B) {
	container := New()
	container.Set(new(iInterface), derive{})
	container.SetWithName(new(iInterface), "custom", derive{})

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var d composeTest
		container.Inject(&d)
	}
}

// BenchmarkSet 度量注册一个实例（含接口断言）的开销。
func BenchmarkSet(b *testing.B) {
	container := New()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		container.Set(new(iInterface), derive{})
	}
}
