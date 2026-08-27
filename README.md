# core

全仓最底层公共库：**零第三方依赖**的原语与通用工具，供所有模块（含 `framework` 与各 leaf 模块）复用。

## 定位与边界

- **只放**零依赖且被多个模块共用的东西：纯 stdlib 原语、通用工具函数、通用接口契约。
- **不放**：依赖第三方（gin/zap/redis/gorm/otel）的能力、IOC 容器、业务/领域能力、单模块私有实现。
- 依赖方向：`core → 无`；`framework → core`；`lock/idempotent/ratelimit/circuitbreaker/cache → core`；其余模块经 `framework` 传递获得。

## 目录结构（对齐 framework）

| 目录 | 用途 | 状态 |
|------|------|------|
| `util/`     | 零依赖纯工具（具体类型/函数，无接口） | ✅ 已建 |
| `contract/` | 全仓通用接口契约（有真正多实现/IOC 需求时再建） | 预留 |
| `service/`  | 契约实现 | 预留 |
| `config/`   | 通用配置结构 | 预留 |
| `entity/`   | 通用实体 | 预留 |
| `enum/`     | 通用枚举 | 预留 |
| `mocks/`    | 契约 mock | 预留 |

## 已提供能力

### goroutine（`util/goroutine.go`）

- `Go(fn)` — 安全起协程：非阻塞直起，内部 recover panic，不拖垮进程。
- `RunLimit(n, tasks)` — 有界并发扇出：最多 n 个同时运行、其余等待，等全部完成返回（等价 `errgroup.SetLimit`）。
- `SetPanicHandler(h)` — 注入 panic 日志处理（默认标准库 log）。

> 设计说明：Go 惯用法不设全局池，而在扇出点限并发（`RunLimit`）；`Go` 只负责安全直起。纯原语、单一实现、不走 IOC，故不定义接口（与 `framework/util` 的 paginator/validator 同类）。

### random（`util/random.go`）

- `RandomHex(n)` — n 字节随机数的十六进制串（长度 2n），基于 crypto/rand，不可预测。
- `UUIDv4()` — RFC 4122 v4 版本 UUID。

## 发布

`git init -b main → gh repo create --public → tag v0.1.0 → push`（`GOPRIVATE` 已设）。
