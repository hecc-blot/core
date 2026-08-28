# core

全仓最底层公共库：纯 stdlib 原语 + 通用接口契约 + 默认实现（日志 zap），供所有模块（含 `framework` 与各 leaf 模块）复用。

## 定位与边界

- **契约层**（`contract/`/`util/`/`config/`/`enum/`）保持零第三方依赖，供 leaf 模块安全复用（只 import 契约不拉第三方）。
- **实现层**（`service/log`）可引第三方：默认本地日志实现用 zap + lumberjack。
- **不放**：依赖 gin/redis/gorm/otel 的能力、IOC 容器、业务/领域能力、单模块私有实现。
- 依赖方向：`core → 无`（仅 `service/log` 引 zap）；`framework → 无`；`log-sls`/`httpclient → core`；`scheduler → core+trace`；`db`/`db-es`/`db-mongo`/`db-clickhouse`/`trace → core+framework`；`mq → core+framework+trace`；`lock → core`（util.RandomHex）；`cache → framework+trace`；`sse`/`metrics → framework`；`ratelimit`/`idempotent`/`circuitbreaker → 第三方`。

## 目录结构（对齐 framework）

| 目录 | 用途 | 状态 |
|------|------|------|
| `util/`     | 零依赖纯工具（具体类型/函数，无接口） | ✅ 已建 |
| `contract/` | 全仓通用接口契约（有真正多实现/IOC 需求时再建） | ✅ 已建（`log`） |
| `service/`  | 契约实现 | ✅ 已建（`log`） |
| `config/`   | 通用配置结构 | ✅ 已建（`log`） |
| `entity/`   | 通用实体 | 预留 |
| `enum/`     | 通用枚举 | ✅ 已建（`trace`） |
| `mocks/`    | 契约 mock | ✅ 已建（`log`） |

## 已提供能力

### goroutine（`util/goroutine.go`）

- `Go(fn)` — 安全起协程：非阻塞直起，内部 recover panic，不拖垮进程。
- `RunLimit(n, tasks)` — 有界并发扇出：最多 n 个同时运行、其余等待，等全部完成返回（等价 `errgroup.SetLimit`）。
- `SetPanicHandler(h)` — 注入 panic 日志处理（默认标准库 log）。

> 设计说明：Go 惯用法不设全局池，而在扇出点限并发（`RunLimit`）；`Go` 只负责安全直起。纯原语、单一实现、不走 IOC，故不定义接口（与 `framework/util` 的 paginator/validator 同类）。

### random（`util/random.go`）

- `RandomHex(n)` — n 字节随机数的十六进制串（长度 2n），基于 crypto/rand，不可预测。
- `UUIDv4()` — RFC 4122 v4 版本 UUID。

### 日志契约（`contract/log`）

- `ILog` — 全仓通用日志接口（Debug/Error/Info/Warn），零依赖，已从 `framework/contract/log` 下沉至此，供 leaf 模块直接复用。

### 日志实现（`service/log`）

- `NewLogger(*config.LocalConfig)` — 本地 zap 默认实现，配合 `config/log`（`LocalConfig`）、`enum/trace`（`TraceIdKey`）使用。
- 厂商后端（SLS）见 `log-sls` 模块，二者通过 `ILog` 契约二选一；业务方不满意默认实现可自行注入。

### 日志配置与使用

本地日志配置（`LocalConfig`，`core/config/log`）：

```yaml
log:
  local:
    root_dir: runtime/logs
    show_line: true        # 是否输出 caller 行号
    max_size: 100          # 单文件最大 100MB
    max_backups: 30        # 最多保留 30 个旧文件
    max_age: 7             # 日志保留 7 天
    compress: true         # 压缩旧文件
```

日志按级别分文件滚动输出：

| 文件 | 级别 | 说明 |
|------|------|------|
| `debug.log` | Debug | 调试信息 |
| `info.log` | Info | 一般信息 |
| `warn.log` | Warn | 警告信息 |
| `error.log` | Error | 错误信息（含 stacktrace） |
| `panic.log` | Panic | 恐慌信息（含 stacktrace） |

使用：

```go
logSvc, _ := log.NewLogger(&config.Log.Local)
logSvc.Info(ctx, "user login success", "ip", c.ClientIP())
logSvc.Error(ctx, "database connection failed", zap.Error(err))

// fields 支持 zapcore.Field 类型
logSvc.Info(ctx, "order created", zap.String("order_id", orderID), zap.Int("amount", 100))
```

从 Context 自动提取 TraceId（key 见 `core/enum/trace`），输出自动包含 `traceId` 和 `caller` 字段：

```json
{
    "level": "info",
    "msg": "user login",
    "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
    "caller": "example/example.go:209",
    "user_id": 123
}
```

## 发布

`git init -b main → gh repo create --public → tag v0.1.0 → push`（`GOPRIVATE` 已设）。
