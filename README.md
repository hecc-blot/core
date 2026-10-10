# hecc-blot-core

全仓最底层公共库：零依赖原语 + 通用接口契约 + 默认实现（日志 zap、IOC 容器），供所有模块（含 `framework` 与各 leaf 模块）复用。

## 安装

```bash
go get github.com/hecc-blot/core
```

## 快速开始

core 对外主要提供**日志**与 **IOC 容器**两类能力，用法都是「先构造、再注册进容器」：

```go
import (
	"context"

	logContract "github.com/hecc-blot/core/contract/log"
	ioc "github.com/hecc-blot/core/service/ioc"
	log "github.com/hecc-blot/core/service/log"

	"go.uber.org/zap"
)

func main() {
	// 1. 构造日志（全仓第一个构造的组件）
	logSvc, err := log.NewLogger(&config.Log.Local)
	if err != nil {
		panic(err)
	}

	// 2. 建容器并把日志注册进去
	container := ioc.New()
	container.Set(new(logContract.ILog), logSvc)

	// 3. 使用
	logSvc.Info(context.Background(), "server starting", zap.String("port", "8080"))
}
```

> 完整的应用组装顺序（trace → db → cache → HTTP 内核 → 路由 → Listen）见[组织主页](https://github.com/hecc-blot)，各模块 README 的「快速开始」都是它的片段。

## 定位与边界

- **契约层**（`contract/`/`util/`/`config/`/`enum/`）保持**零第三方依赖**，供 leaf 模块安全复用（只 import 契约不拉第三方）。
- **实现层**（`service/log`）可引第三方：默认本地日志实现用 zap + lumberjack。
- **不放**：依赖 gin/redis/gorm/otel 的能力、业务/领域能力、单模块私有实现。
- **IOC 容器**（`contract/ioc` + `service/ioc` + `mocks/ioc`）：纯 stdlib（`reflect` + `fmt`）、零第三方依赖，是全仓最底层的通用组装能力，故置于此。它与「不放」清单里 gin/redis/gorm 那类的区别在于——那些会往下游拉第三方，IOC 不会。
- 依赖方向：`core → 无`（仅 `service/log` 引 zap）；`framework → core`；`log-sls`/`httpclient`/`lock → core`；`scheduler`/`mq → core+trace`；`db`/`db-es`/`db-mongo`/`db-clickhouse`/`trace → core+framework`；`cache → framework+trace`；`sse → framework+core`；`metrics → framework`；`ratelimit`/`idempotent`/`circuitbreaker → 第三方`。

## 包结构

| 目录 | 用途 | 内容 |
|------|------|------|
| `util/`     | 零依赖纯工具（具体类型/函数，无接口） | `goroutine` / `random` / `ExtractTraceId` |
| `contract/` | 全仓通用接口契约 | `log`、`ioc` |
| `service/`  | 契约实现 | `log`（zap + lumberjack）、`ioc`（反射容器） |
| `config/`   | 通用配置结构 | `log` |
| `entity/`   | 通用实体 | 预留 |
| `enum/`     | 通用枚举 | `trace`（`TraceIdKey`） |
| `mocks/`    | 契约 mock | `log`、`ioc` |

## 接口定义

```go
import (
	logContract "github.com/hecc-blot/core/contract/log"
	iocContract "github.com/hecc-blot/core/contract/ioc"
)

// 日志契约
type ILog interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
}

// 依赖注入容器契约。框架组件（ApiHandle / SseHandle / ConsumerHandle）依赖此接口而非具体实现，
// 业务方可替换为自己的容器。
type IContainer interface {
	Get(interfaceObj interface{}, name string) interface{}
	Inject(instance interface{})                        // instance 必须是指针
	Set(interfaceObj interface{}, instance interface{})
	SetWithName(interfaceObj interface{}, name string, instance interface{})
}
```

## 使用

### 日志（`service/log`）

```go
logSvc, err := log.NewLogger(&config.Log.Local)
if err != nil {
	panic(err)
}

logSvc.Info(ctx, "user login success", zap.String("ip", c.ClientIP()))
logSvc.Error(ctx, "database connection failed", zap.Error(err))
```

> **`fields` 只认 `zapcore.Field`**。实现只保留 `zap.String(...)` / `zap.Int(...)` / `zap.Error(...)` 这类字段，其它类型（包括裸的 `"ip", value` 这种键值对）会被**静默丢弃**——不报错，也不会出现在日志里。

日志会自动从 `ctx` 提取 traceId（key 见 `core/enum/trace`），输出包含 `traceId` 与 `caller`：

```json
{
    "level": "info",
    "msg": "user login",
    "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
    "caller": "main.go:209",
    "user_id": 123
}
```

日志按级别分文件滚动输出：

| 文件 | 级别 | 说明 |
|------|------|------|
| `debug.log` | Debug | 调试信息 |
| `info.log` | Info | 一般信息 |
| `warn.log` | Warn | 警告信息 |
| `error.log` | Error | 错误信息（含 stacktrace） |
| `panic.log` | Panic | 恐慌信息（含 stacktrace） |

### IOC 容器（`service/ioc`）

```go
container := ioc.New()

// 初始化阶段注册（单线程，先注册后启动）
container.Set(new(logContract.ILog), logSvc)
container.SetWithName(new(logContract.ILog), "remote", remoteLog) // 同接口多实现用命名注册

// 运行阶段注入：字段带 inject:"" 才会被注入，且必须放在最前面
type AddApi struct {
	LogSvc    logContract.ILog `inject:""`
	AddRequest                 // 请求参数放在最后
}
container.Inject(&AddApi{})
```

> **注入规则（最高优先级约定）**：字段带 `inject:""` tag 才会被注入，遍历时**遇到第一个没有 `inject` tag 的具名字段就停止**（`service/ioc/ioc_svc.go` 里的 `return`，是刻意设计不是 bug）。因此注入字段必须放在最前面，请求参数等放在最后。匿名嵌套的非 struct 字段（如 `sync.Mutex`）会 `continue`，不中断扫描。

### goroutine（`util/goroutine.go`）

```go
util.Go(ctx, func() { doSomething() })                 // 安全起协程：内部 recover panic，不拖垮进程
util.RunLimit(ctx, 4, []func(){taskA, taskB, taskC})   // 有界并发扇出：最多 4 个同时跑，全部完成才返回
util.SetPanicHandler(myHandler)                       // 注入 panic 处理（默认走标准库 log），可随时调、并发安全
```

`ctx` 只用于 panic 归因（处理器里能取到 traceId），**不控制任务生命周期** —— 需要取消/超时请在任务内自行监听 `ctx.Done()`；传 `nil` 会归一为 `Background`。

处理器签名带 ctx，所以 panic 日志可以关联到请求：

```go
util.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
    logger.Error(ctx, "goroutine panic",
        zap.Any("panic", r), zap.String("stack", string(stack)))
})
```

> `SetPanicHandler` 并发安全，启动期调一次即可，之后也可替换；传 `nil` 是 no-op（保持当前值，不会退回默认）。

> 设计说明：Go 惯用法不设全局池，而在扇出点限并发（`RunLimit`）；`Go` 只负责安全直起。

### random（`util/random.go`）

```go
util.RandomHex(16)  // 16 字节随机数的十六进制串（长度 32），基于 crypto/rand
util.UUIDv4()       // RFC 4122 v4 UUID
```

### traceId 提取（`util/trace.go`）

```go
traceId := util.ExtractTraceId(ctx)   // 取不到返回 ""
```

从 `ctx` 里读 `enum/trace.TraceIdKey`。**前提是 ctx 的 `Value` 能查到该 key**：经 HTTP 内核传入的 `IContext` 依赖 `NewApiSvc` 开启的 gin `ContextWithFallback`（见 framework README）。

## 配置

```yaml
log:
  local:
    root_dir: runtime/logs   # 日志根目录（相对进程工作目录，不存在会自动创建）
    max_size: 100            # 单文件最大 100MB
    max_backups: 30          # 最多保留 30 个旧文件
    max_age: 7               # 日志保留 7 天
    compress: true           # 压缩旧文件
```

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `root_dir` | string | — | 日志根目录 |
| `max_size` | int | — | 单文件上限（MB） |
| `max_backups` | int | — | 保留的旧文件数 |
| `max_age` | int | — | 保留天数 |
| `compress` | bool | — | 是否压缩旧文件 |

> `LocalConfig` 还有一个 `ShowLine` 字段，但**当前实现恒开启 caller 输出、不读该字段**，配置它没有效果。

## 语义与约定

- **日志**：`NewLogger` 会创建 `root_dir`；四个级别都并发安全，可全局共享一个实例。
- **容器并发**：`Container` 内部 map **不加锁**——`Set`/`SetWithName` 仅允许在启动初始化阶段调用；初始化完成后容器只读，`Get`/`Inject` 可安全并发；**运行时禁止再 `Set`**。
- **容器 panic 条件**：
  - `Inject` 传入非指针 → panic（`ioc: 注入实例必须是指针`）
  - `Set` 的实例未实现目标接口 → panic（`ioc: %v没有实现%v`）
  - 注入时依赖**未注册** → panic（`ioc: 无效类型(...)`）——这是最常见的踩坑点
- **无释放语义**：core 不持有需要关闭的资源（日志文件由 lumberjack 管理），不需要 cleanup。

## 相关模块

| 模块 | 说明 |
|------|------|
| [framework](https://github.com/hecc-blot/framework) | 依赖 core 的 `IContainer` 与 `ILog` |
| [log-sls](https://github.com/hecc-blot/log-sls) | `ILog` 的阿里云 SLS 实现，与本地日志二选一 |
| [trace](https://github.com/hecc-blot/trace) | 写入 `TraceIdKey`，被 core 日志读取 |

框架整体组装顺序见[组织主页](https://github.com/hecc-blot)。
