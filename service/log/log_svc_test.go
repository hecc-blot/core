package service

import (
	"context"
	"testing"

	logConfig "github.com/hecc-blot/core/config/log"
	"github.com/hecc-blot/core/enum/trace"
)

var localConf = &logConfig.LocalConfig{
	RootDir:    "./runtime/logs",
	MaxSize:    1,
	MaxBackups: 3,
	MaxAge:     7,
	Compress:   false,
}

func TestLogSvc(t *testing.T) {
	logger, err := newLogSvc(localConf)
	if err != nil {
		t.Fatalf("newLogSvc 失败: %v", err)
	}
	if logger == nil {
		t.Fatal("logger 不应为 nil")
	}

	ctx := context.Background()

	cases := []struct {
		level string
		text  string
	}{
		{"debug", "test debug"},
		{"info", "test info"},
		{"warn", "test warn"},
		{"error", "test error"},
	}
	t.Run("log", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.level, func(t *testing.T) {
				switch c.level {
				case "debug":
					logger.Debug(ctx, c.text)
				case "info":
					logger.Info(ctx, c.text)
				case "warn":
					logger.Warn(ctx, c.text)
				case "error":
					logger.Error(ctx, c.text)
				}
			})
		}
	})
}

func TestLogSvcWithTrace(t *testing.T) {
	logger, err := newLogSvc(localConf)
	if err != nil {
		t.Fatalf("newLogSvc 失败: %v", err)
	}

	ctxWithTraceId := context.WithValue(context.Background(), trace.TraceIdKey, "test-trace-id")

	cases := []struct {
		level string
		text  string
	}{
		{"debug", "test debug"},
		{"info", "test info"},
		{"warn", "test warn"},
		{"error", "test error"},
	}

	t.Run("log with trace", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.level, func(t *testing.T) {
				switch c.level {
				case "debug":
					logger.Debug(ctxWithTraceId, c.text)
				case "info":
					logger.Info(ctxWithTraceId, c.text)
				case "warn":
					logger.Warn(ctxWithTraceId, c.text)
				case "error":
					logger.Error(ctxWithTraceId, c.text)
				}
			})
		}
	})
}
