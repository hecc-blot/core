package service

import (
	"context"
	"os"
	"time"

	logConfig "github.com/hecc-blot/core/config/log"
	logContract "github.com/hecc-blot/core/contract/log"
	"github.com/hecc-blot/core/util"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogSvc struct {
	logger *zap.Logger
}

func (b LogSvc) Debug(ctx context.Context, msg string, fields ...interface{}) {
	b.logger.Debug(msg, b.buildFields(util.ExtractTraceId(ctx), fields...)...)
}

func (b LogSvc) Error(ctx context.Context, msg string, fields ...interface{}) {
	b.logger.Error(msg, b.buildFields(util.ExtractTraceId(ctx), fields...)...)
}

func (b LogSvc) Info(ctx context.Context, msg string, fields ...interface{}) {
	b.logger.Info(msg, b.buildFields(util.ExtractTraceId(ctx), fields...)...)
}

func (b LogSvc) Warn(ctx context.Context, msg string, fields ...interface{}) {
	b.logger.Warn(msg, b.buildFields(util.ExtractTraceId(ctx), fields...)...)
}

func (b LogSvc) buildFields(traceId string, fields ...interface{}) []zapcore.Field {
	zapFields := make([]zapcore.Field, 0, len(fields)+1)

	if traceId != "" {
		zapFields = append(zapFields, zap.String("traceId", traceId))
	}

	for _, v := range fields {
		if field, ok := v.(zapcore.Field); ok {
			zapFields = append(zapFields, field)
		}
	}

	return zapFields
}

// NewLogger 创建本地日志服务（core 默认 logger，zap 文件滚动）。
// SLS 等厂商后端见 github.com/hecc-blot/log-sls/service，由组装层按配置二选一。
func NewLogger(logConf *logConfig.LocalConfig) (logContract.ILog, error) {
	var rootDir string
	var maxSize, maxBackups, maxAge int
	var compress bool

	levels := map[zapcore.Level]string{
		zapcore.DebugLevel: "debug",
		zapcore.InfoLevel:  "info",
		zapcore.WarnLevel:  "warn",
		zapcore.ErrorLevel: "error",
		zapcore.PanicLevel: "panic",
	}

	rootDir = logConf.RootDir
	maxSize = logConf.MaxSize
	maxBackups = logConf.MaxBackups
	maxAge = logConf.MaxAge
	compress = logConf.Compress

	// 判断目录是否存在
	_, err := os.Stat(rootDir)
	if os.IsNotExist(err) {
		// 创建根目录
		if err = os.MkdirAll(rootDir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// 扩展Zap
	var encoder zapcore.Encoder

	// 调整编码器默认配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(time.Format("[" + "2006-01-02 15:04:05.000" + "]"))
	}
	encoderConfig.EncodeLevel = func(l zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(l.String())
	}

	// 设置编码器
	encoder = zapcore.NewJSONEncoder(encoderConfig)

	var cores []zapcore.Core
	for level, suffix := range levels {
		cores = append(cores, zapcore.NewCore(
			encoder,
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   rootDir + "/" + suffix + ".log",
				MaxSize:    maxSize,
				MaxBackups: maxBackups,
				MaxAge:     maxAge,
				Compress:   compress,
			}),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl == level
			}),
		))
	}

	return LogSvc{
		logger: zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel)),
	}, nil
}
