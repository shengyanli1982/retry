<div align="center">
    <img src="assets/logo.png" alt="logo" width="500px">
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/shengyanli1982/retry)](https://goreportcard.com/report/github.com/shengyanli1982/retry)
[![Build Status](https://github.com/shengyanli1982/retry/actions/workflows/test.yaml/badge.svg)](https://github.com/shengyanli1982/retry/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shengyanli1982/retry.svg)](https://pkg.go.dev/github.com/shengyanli1982/retry)

A lightweight, zero-dependency retry library for Go. Retry wraps any function with configurable backoff, jitter, per-error budgets, and context-aware cancellation — all in a single fluent call.

**Features:**

- **Configurable retries** — total attempts + per-error budgets
- **Composable backoff** — exponential, random, fixed, or any combination
- **Jitter & factor** — randomized linear delay to prevent thundering herd
- **Callback hooks** — `OnRetry` fires before each actual retry
- **Context-aware** — respects `context.Context` cancellation and deadlines
- **Zero dependencies** — standard library only
- **Concurrency-safe** — safe for concurrent use with shared `Retry` instances

**Requires Go 1.23+.**

## Install

```bash
go get github.com/shengyanli1982/retry
```

## Quick Start

```go
import "github.com/shengyanli1982/retry"

result := retry.DoWithDefault(func() (any, error) {
    return callExternalService()
})

if result.IsSuccess() {
    fmt.Println(result.Data())
}
```

## How It Works

### Execution Flow

```
    fn() ──► success? ──► return Result
               │ no
               ▼
         retryIf allows?
           │         │
          yes        no ──► abort (ErrorRetryIf)
           │
           ▼
     budget available?
       │          │
      yes         no ──► abort (ErrorRetryAttemptsExceeded)
       │
       ▼
   wait backoff ──► fn() ──► ...
```

1. Execute `fn` immediately (no timer allocated on the success path).
2. On error: check `retryIfFunc` → check per-error budget → check total attempts.
3. If all checks pass, compute backoff delay, fire `OnRetry` callback, then wait and retry.
4. Repeat until success, budget exhaustion, or context cancellation.

### Backoff Formula

```
delay = backoffFunc(count) + trunc(jitter × rand.Float64() + factor × count) × initDelay
```

| Component       | Default                            | Description                                      |
| --------------- | ---------------------------------- | ------------------------------------------------ |
| `backoffFunc`   | `Exponential + Random`             | Base backoff in 100ms units                      |
| `initDelay`     | `500ms`                            | Multiplier base for the linear delay component   |
| `factor`        | `1.0`                              | Linear growth per retry attempt                  |
| `jitter`        | `3.0`                              | Random [0,1) multiplier for the linear component |

`count` is the number of completed executions (starting from 1). The linear term is truncated to `time.Duration` (toward zero), quantizing it to integer multiples of `initDelay`.

### Config Normalization

| Field       | Invalid Input    | Behavior                        |
| ----------- | ---------------- | ------------------------------- |
| `attempts`  | `0`              | Reset to default `3`            |
| `attempts`  | `>= 65535`       | Clamped to `65534`              |
| `delay`     | `<= 0`           | Reset to default `500ms`        |
| `factor`    | `< 0`            | Reset to default `1.0`          |
| `jitter`    | `< 0`            | Reset to default `3.0`          |
| `ctx`       | `nil`            | Set to `context.Background()`   |
| `callback`  | `nil`            | Set to no-op callback           |

`New()` shallow-copies the config — normalization never mutates the caller's object.

## API Reference

### Entry Points

| Function                          | Description                                                              |
| --------------------------------- | ------------------------------------------------------------------------ |
| `Do(fn, conf)`                    | Execute with config; `nil` config means defaults                         |
| `DoWithDefault(fn)`               | Execute with all defaults; equivalent to `Do(fn, nil)`                   |
| `New(conf) *Retry`                | Create reusable instance; config is shallow-copied                       |
| `(*Retry).TryOnConflict(fn)`      | Execute and retry; returns `*Result` (`nil` if `fn` is `nil`)           |
| `(*Retry).TryOnConflictVal(fn)`   | Same as above, returns `RetryResult` interface (true `nil` if `fn` nil) |

All entry points return a true `nil` interface (not a typed nil) when `fn` is `nil`.

### Config Builders

| Method                      | Default              | Description                              |
| --------------------------- | -------------------- | ---------------------------------------- |
| `WithContext(ctx)`          | `context.Background` | Cancellation and deadline control        |
| `WithCallback(cb)`          | No-op                | `OnRetry` hook                           |
| `WithAttempts(n)`           | `3`                  | Total execution budget                   |
| `WithAttemptsByError(map)`  | —                    | Per-error execution budget (map is copied) |
| `WithInitDelay(d)`          | `500ms`              | Linear delay multiplier base             |
| `WithFactor(f)`             | `1.0`                | Linear growth per attempt                |
| `WithJitter(j)`             | `3.0`                | Random delay multiplier                  |
| `WithRetryIfFunc(fn)`       | Always `true`        | Condition to allow retry                 |
| `WithBackOffFunc(fn)`       | `Exp + Random`       | Backoff strategy                         |
| `WithDetail(bool)`          | `false`              | Record every execution error             |

### Result Interface

| Method                 | Description                                                              |
| ---------------------- | ------------------------------------------------------------------------ |
| `Data() any`           | Return value on success                                                  |
| `TryError() error`     | Final abort error (`nil` on success)                                     |
| `ExecErrors() []error` | All recorded errors (only when `detail=true`)                            |
| `IsSuccess() bool`     | Whether the execution succeeded                                          |
| `LastExecError()`      | Last execution error, or `ErrorExecErrNotFound`                          |
| `FirstExecError()`     | First execution error, or `ErrorExecErrNotFound`                         |
| `ExecErrorByIndex(i)`  | Error at index, or `ErrorExecErrByIndexOutOfBound`                       |
| `Count() int64`        | Total number of executions performed                                     |

### Backoff Functions

`BackoffFunc` signature: `func(count int64) time.Duration` — all built-in functions use 100ms as the base unit.

| Function                       | Formula                    | Guard                             |
| ------------------------------ | -------------------------- | --------------------------------- |
| `FixedBackoff(interval)`       | `interval × 100ms`        | `interval <= 0` → `500ms`        |
| `RandomBackoff(maxInterval)`   | `rand[0, max) × 100ms`    | `maxInterval <= 0` → `500ms`     |
| `ExponentialBackoff(power)`    | `2^power × 100ms`         | `power <= 0` → `500ms`; clamped at `36` |
| `CombineBackoffs(fns...)`      | `sum(fn(count))`          | empty → `FixedBackoff`; `<= 0` → `500ms` |

**Default:** `CombineBackoffs(ExponentialBackoff, RandomBackoff)`.

All backoff functions clamp inputs to prevent `time.Duration` overflow (`2^36 × 100ms < math.MaxInt64`; `2^37` wraps negative).

### Sentinel Errors

Every abort path wraps the sentinel together with the root cause via `wrappedError` (zero-allocation `Unwrap() []error`), so `errors.Is` matches both:

| Sentinel                                | Triggered When                                   |
| --------------------------------------- | ------------------------------------------------ |
| `ErrorRetryIf`                          | `retryIfFunc` returns `false`                    |
| `ErrorRetryAttemptsExceeded`            | Total `attempts` budget exhausted                |
| `ErrorRetryAttemptsByErrorExceeded`     | Per-error budget exhausted                       |
| `ErrorExecErrNotFound`                  | `LastExecError`/`FirstExecError` with no records |
| `ErrorExecErrByIndexOutOfBound`         | `ExecErrorByIndex` with out-of-range index       |

```go
result := retry.DoWithDefault(fn)
if errors.Is(result.TryError(), retry.ErrorRetryAttemptsExceeded) {
    // total budget exhausted; root cause also reachable via errors.Is
}
```

### Callback

`OnRetry` is called only right before a retry is actually initiated — after all abort checks pass. It is **not** called when the final attempt fails and the retry aborts. With `attempts = 3` and every execution failing, `OnRetry` fires exactly twice.

```go
type Callback interface {
    OnRetry(count int64, delay time.Duration, err error)
}
```

### Convenience Configs

| Function          | Description                                            |
| ----------------- | ------------------------------------------------------ |
| `NewConfig()`     | Default config (same as `DefaultConfig()`)             |
| `FixConfig()`     | Fixed `500ms` interval (`factor=0`, `jitter=0`)       |

## Usage Examples

### Basic Retry

```go
result := retry.DoWithDefault(func() (any, error) {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return resp, nil
})

if !result.IsSuccess() {
    log.Fatal(result.TryError())
}
data := result.Data().(*http.Response)
```

### Custom Config

```go
cfg := retry.NewConfig().
    WithAttempts(5).
    WithInitDelay(200 * time.Millisecond).
    WithJitter(2.0).
    WithFactor(0.5).
    WithDetail(true).
    WithRetryIfFunc(func(err error) bool {
        return !errors.Is(err, context.Canceled)
    })

result := retry.Do(func() (any, error) {
    return callService()
}, cfg)

for i, e := range result.ExecErrors() {
    fmt.Printf("attempt %d: %v\n", i+1, e)
}
```

### Per-Error Budget

```go
transientErr := errors.New("transient")
permanentErr := errors.New("permanent")

cfg := retry.NewConfig().
    WithAttemptsByError(map[error]uint64{
        transientErr: 5,
        permanentErr: 0, // never retry
    })

result := retry.Do(func() (any, error) {
    return callService()
}, cfg)

if errors.Is(result.TryError(), retry.ErrorRetryAttemptsByErrorExceeded) {
    // per-error budget exhausted
}
```

### Callback

```go
type logger struct{}

func (l *logger) OnRetry(count int64, delay time.Duration, err error) {
    fmt.Printf("retry #%d after %v: %v\n", count, delay, err)
}

cfg := retry.NewConfig().WithCallback(&logger{}).WithAttempts(3)
result := retry.Do(func() (any, error) {
    return callService()
}, cfg)
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cfg := retry.NewConfig().WithContext(ctx)
result := retry.Do(func() (any, error) {
    return callService()
}, cfg)

if errors.Is(result.TryError(), context.DeadlineExceeded) {
    // context timed out before success
}
```

### Factory Pattern

Reuse a single `Retry` instance across multiple functions:

```go
r := retry.New(retry.NewConfig().WithAttempts(5))

result1 := r.TryOnConflict(funcA)
result2 := r.TryOnConflict(funcB)
```

## Concurrency

`Retry` is safe for concurrent use. Multiple goroutines can call `TryOnConflict` on the same instance simultaneously.

**Design details:**

- `New()` shallow-copies the config, so concurrent callers never race on normalization
- `WithAttemptsByError()` copies the user map at entry, preventing concurrent map access
- `attemptsByError` is cloned per-call inside `TryOnConflict` to avoid shared map reads
- `rand/v2` top-level functions are goroutine-safe and per-M lock-free
- Subscriber callbacks (`OnRetry`) execute synchronously — do not block inside them

## Benchmarks

```
$ go test -bench=. -benchmem -count=5
BenchmarkNew                                0 allocs/op      0 B/op
BenchmarkNew_NilConfig                      1 allocs/op     64 B/op
BenchmarkTryOnConflict_Success              2 allocs/op    112 B/op
BenchmarkTryOnConflict_Retry3               6 allocs/op    320 B/op
BenchmarkTryOnConflict_AllFail              5 allocs/op    288 B/op
BenchmarkTryOnConflict_WithDetail           6 allocs/op    416 B/op
BenchmarkTryOnConflict_WithAttemptsByError  5 allocs/op    288 B/op
BenchmarkDo                                 2 allocs/op    112 B/op
BenchmarkDoWithDefault                      3 allocs/op    176 B/op
BenchmarkFixedBackoff                       0 allocs/op      0 B/op
BenchmarkRandomBackoff                      0 allocs/op      0 B/op
BenchmarkExponentialBackoff                 0 allocs/op      0 B/op
BenchmarkCombinedBackoffs                   0 allocs/op      0 B/op
```

Success path allocates only 2 objects (`Result` + shallow `Config` copy). Backoff functions are zero-alloc.

## Examples

See the [`examples/`](./examples/) directory for runnable demos covering all features.

## License

[MIT](./LICENSE)
