package mocks

import "context"

// MockLog 是 ILog 接口的 mock 实现，记录各日志级别的调用消息，供单测断言。
type MockLog struct {
	InfoMsgs  []string
	DebugMsgs []string
	WarnMsgs  []string
	ErrorMsgs []string
}

func (m *MockLog) Info(ctx context.Context, msg string, fields ...interface{}) {
	m.InfoMsgs = append(m.InfoMsgs, msg)
}

func (m *MockLog) Debug(ctx context.Context, msg string, fields ...interface{}) {
	m.DebugMsgs = append(m.DebugMsgs, msg)
}

func (m *MockLog) Warn(ctx context.Context, msg string, fields ...interface{}) {
	m.WarnMsgs = append(m.WarnMsgs, msg)
}

func (m *MockLog) Error(ctx context.Context, msg string, fields ...interface{}) {
	m.ErrorMsgs = append(m.ErrorMsgs, msg)
}
