# core

全仓最底层公共库：纯 stdlib 原语 + 通用接口契约 + 默认实现（日志 zap），供所有模块（含 `framework` 与各 leaf 模块）复用。

## 定位与边界

- **契约层**（`contract/`/`util/`/`config/`/`enum/`）保持零第三方依赖，供 leaf 模块安全复用（只 import 契约不拉第三方）。
- **实现层**（`service/log`）可引第三方：默认本地日志实现用 zap + lumberjack。
- **不放**：依赖 gin/redis/gorm/otel 的能力、业务/领域能力、单模块私有实现。
- **IOC 容器**（`contract/ioc` + `service/ioc` + `mocks/ioc`）：纯 stdlib（`reflect` + `fmt`）、零第三方依赖，是全仓最底层的通用组装能力，故置于此。它与「不放」清单里 gin/redis/gorm 那类的区别在于——那些会往下游拉第三方，IOC 不会。已从 `framework` 下沉至此，供 `framework`/`sse`/`mq` 直接复用。
- 依赖方向：`core → 无`（仅 `service/log` 引 zap）；`framework → core`；`log-sls`/`httpclient`/`lock → core`；`scheduler`/`mq → core+trace`；`db`/`db-es`/`db-mongo`/`db-clickhouse`/`trace → core+framework`；`cache → framework+trace`；`sse → framework+core`；`metrics → framework`；`ratelimit`/`idempotent`/`circuitbreaker → 第三方`。

## 目录结构（对齐 framework）

| 目录 | 用途 | 状态 |
|------|------|------|
| `util/`     | 零依赖纯工具（具体类型/函数，无接口） | ✅ 已建 |
| `contract/` | 全仓通用接口契约 | ✅ 已建（`log`、`ioc`） |
| `service/`  | 契约实现 | ✅ 已建（`log`、`ioc`） |
| `config/`   | 通用配置结构 | ✅ 已建（`log`） |
| `entity/`   | 通用实体 | 预留 |
| `enum/`     | 通用枚举 | ✅ 已建（`trace`） |
| `mocks/`    | 契约 mock | ✅ 已建（`log`、`ioc`） |

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

### IOC 容器（`contract/ioc` + `service/ioc` + `mocks/ioc`）

- `IContainer` — 依赖注入容器契约（`Get` / `Inject` / `Set` / `SetWithName`），零依赖。`framework` 的 `ApiHandle`、`sse` 的 `SseHandle`、`mq` 的 `ConsumerHandle` 均依赖此接口而非具体实现，业务方可替换为自己的容器。
- `New()` — 默认反射实现，可实例化、支持多容器隔离，纯 stdlib。
- `mocks.MockContainer` — 契约 mock，供单测复用。

```go
container := ioc.New()
container.Set(new(dbContract.IDbFactory), dbFactory)   // 初始化阶段注册
container.Inject(&someApi)                              // 运行阶段注入
```

> **注入规则（最高优先级约定）**：字段带 `inject:""` tag 才会被注入，且遍历时**遇到第一个没有 `inject` tag 的具名字段就停止**（`service/ioc/ioc_svc.go` 里的 `return`，是刻意设计不是 bug）。因此注入字段必须放在最前面，请求参数等放在最后。匿名嵌套的非 struct 字段（如 `sync.Mutex`）会 `continue`，不中断扫描。

> 该模块原先位于 `framework`，2026-09-12 下沉至此：mq 的消费者注册器需要 `IContainer`，而 mq 不应为此依赖整个 framework；IOC 本身是纯 stdlib 的通用组装能力，本就符合 core 的定位。

## 发布

`git init -b main → gh repo create --public → tag v0.1.0 → push`（`GOPRIVATE` 已设）。
