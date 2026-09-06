[English](./README.md) | 中文

<div align="center">
	<h1>Retry</h1>
	<p>一个简单、无依赖的 Go 函数执行模块，用于在各种场景下轻松进行函数重试。</p>
	<img src="assets/logo.png" alt="logo" width="350px">
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/shengyanli1982/retry)](https://goreportcard.com/report/github.com/shengyanli1982/retry)
[![Build Status](https://github.com/shengyanli1982/retry/actions/workflows/test.yaml/badge.svg)](github.com/shengyanli1982/retry/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shengyanli1982/retry.svg)](https://pkg.go.dev/github.com/shengyanli1982/retry)

# 介绍

`Retry` 是一个轻量级的函数重试模块。它简单易用，没有第三方依赖。它专为需要重试函数调用的场景而设计。

`Retry` 提供以下功能：

1. 可以指定重试函数调用的次数。
2. 可以指定特定错误的重试次数。
3. 支持回调函数。
4. 支持延迟的抖动因子。
5. 支持指数退避延迟、随机延迟和固定延迟。
6. 支持记录每次失败重试的详细错误信息。

# 优势

-   简单易用
-   无需外部依赖
-   内存使用高效
-   支持回调函数

# 安装

```bash
go get github.com/shengyanli1982/retry
```

# 快速入门

使用 `Retry` 很简单。只需要一行代码就可以重试函数调用。

## 1. 普通模式

### 配置

`Retry` 提供了一个配置对象来自定义重试行为。配置对象具有以下字段：

-   `ctx`：上下文对象 `context.Context`。默认值为 `context.Background()`。
-   `callback`：回调函数。默认值为 `&emptyCallback{}`。
-   `attempts`：重试次数。默认值为 `3`。
-   `attemptsByError`：特定错误的重试次数。默认值为 `map[error]uint64{}`。
-   `delay`：基础延迟，是每次重试线性延迟项的乘数基数（见下方退避公式）。默认值为 `500ms`。
-   `factor`：重试次数的因子。默认值为 `1.0`。
-   `jitter`：抖动因子，为线性延迟项添加随机性。默认值为 `3.0`。
-   `retryIfFunc`：确定是否重试的函数。默认值为 `defaultRetryIfFunc`。
-   `backoffFunc`：退避函数。默认值为 `defaultBackoffFunc`。
-   `detail`：是否记录详细错误信息。默认值为 `false`。

您可以使用以下方法来设置配置值：

-   `WithContext`：设置上下文对象 `context.Context`。
-   `WithCallback`：设置回调函数。
-   `WithAttempts`：设置重试次数。取值在传入 `New` 时归一化：`0` 回退为默认值 `3`，`>= 65535` 的值被钳制为 `65534`。
-   `WithAttemptsByError`：设置特定错误的重试次数。
-   `WithInitDelay`：设置基础延迟（每次重试线性延迟项的乘数基数）。
-   `WithJitter`：设置抖动因子。
-   `WithFactor`：设置重试次数的因子。
-   `WithRetryIfFunc`：设置确定是否重试的函数。
-   `WithBackOffFunc`：设置退避函数。
-   `WithDetail`：设置是否记录详细错误信息。

> [!NOTE]
> 退避算法决定了重试之间的延迟时间。`Retry` 支持三种退避算法：指数退避、随机退避和固定退避。默认情况下，`Retry` 使用指数退避与随机退避值之和。
>
> 您可以使用 `WithBackOffFunc` 方法来设置退避算法。
>
> **eg**: backoff = backoffFunc(count) + trunc(jitter \* rand.Float64() + factor \* count) \* delay
>
> 其中 `count` 为已完成的执行次数（从 1 开始），`rand.Float64()` 返回 `[0, 1)` 内的随机值，`trunc` 为 float64 到 `time.Duration` 的转换（向零截断）——因此线性项被量化为 `delay` 的整数倍。退避函数仅接收 `count` 作为入参；内置退避函数以 100ms 基础时间单位为倍数计算结果（见 [API 参考](#api-参考)）。

### 方法

-   `Do`: 通过指定配置对象和函数来重试函数调用。它返回一个 `RetryResult` 接口值（由 `*Result` 实现）。
-   `DoWithDefault`: 使用默认配置值来重试函数调用。它返回一个 `RetryResult` 接口值（由 `*Result` 实现）。

> [!TIP]
> 返回的 `RetryResult` 值内包含函数调用的结果、最后一次重试的错误、所有重试的错误以及重试是否成功。如果函数调用失败，将返回默认值。

### 执行结果

在重试之后，`Retry` 返回一个 `RetryResult` 接口值（由 `*Result` 实现）。`RetryResult` 接口提供以下方法：

-   `Data`: 获取成功调用函数的结果。类型为 `interface{}`。
-   `TryError`: 获取重试操作的错误。如果重试成功，则值为 `nil`。
-   `ExecErrors`: 获取所有重试的错误。仅当 `detail` 为 `true` 时才记录错误，否则列表保持为空。
-   `IsSuccess`: 检查重试操作是否成功。
-   `LastExecError`: 获取最后一次重试的错误。未记录任何错误时返回哨兵错误 `ErrorExecErrNotFound`。
-   `FirstExecError`: 获取第一次重试的错误。未记录任何错误时返回哨兵错误 `ErrorExecErrNotFound`。
-   `ExecErrorByIndex`: 通过索引获取特定重试的错误。索引越界时返回哨兵错误 `ErrorExecErrByIndexOutOfBound`。
-   `Count`: 获取已执行的次数。类型为 `int64`。

### 示例

```go
package main

import (
	"fmt"

	"github.com/shengyanli1982/retry"
)

// 定义一个可重试的函数
// Define a retryable function
func testFunc() (any, error) {
	// 此函数返回一个字符串 "lee" 和一个 nil 错误
	// This function returns a string "lee" and a nil error
	return "lee", nil
}

func main() {
	// 使用默认的重试策略调用 testFunc 函数
	// Call the testFunc function using the default retry strategy
	result := retry.DoWithDefault(testFunc)

	// 打印执行结果
	// Print the execution result
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	// Print the error of the attempt to execute
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	// Print all errors during execution
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	// Print whether the execution was successful
	fmt.Println("isSuccess:", result.IsSuccess())
}
```

**Result**

```bash
$ go run demo.go
result: lee
tryError: <nil>
execErrors: []
isSuccess: true
```

## 2. 工厂模式

工厂模式提供了与普通模式相同的重试函数和功能。它使用相同的 `Config`、`Methods`、`Result` 和 `Callback`。

唯一的区别是使用 `New` 方法创建 `Retry` 对象，然后可以使用 `TryOnConflict` 方法以相同的参数重试函数调用。

### 示例

```go
package main

import (
	"errors"
	"fmt"

	"github.com/shengyanli1982/retry"
)

// 定义一个可重试的函数 testFunc1
// Define a retryable function testFunc1
func testFunc1() (any, error) {
	// 此函数返回一个字符串 "testFunc1" 和一个 nil 错误
	// This function returns a string "testFunc1" and a nil error
	return "testFunc1", nil
}

// 定义一个可重试的函数 testFunc2
// Define a retryable function testFunc2
func testFunc2() (any, error) {
	// 此函数返回一个 nil 和一个新的错误 "testFunc2"
	// This function returns a nil and a new error "testFunc2"
	return nil, errors.New("testFunc2")
}

func main() {
	// 使用默认的配置创建一个新的重试实例
	// Create a new retry instance with the default configuration
	r := retry.New(nil)

	// 尝试执行 testFunc1 函数，如果遇到冲突则进行重试
	// Try to execute the testFunc1 function, retry if there is a conflict
	result := r.TryOnConflict(testFunc1)

	// 打印 testFunc1 执行结果
	// Print the testFunc1 execution result
	fmt.Println("========= testFunc1 =========")

	// 打印执行结果
	// Print the execution result
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	// Print the error of the attempt to execute
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	// Print all errors during execution
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	// Print whether the execution was successful
	fmt.Println("isSuccess:", result.IsSuccess())

	// 尝试执行 testFunc2 函数，如果遇到冲突则进行重试
	// Try to execute the testFunc2 function, retry if there is a conflict
	result = r.TryOnConflict(testFunc2)

	// 打印 testFunc2 执行结果
	// Print the testFunc2 execution result
	fmt.Println("========= testFunc2 =========")

	// 打印执行结果
	// Print the execution result
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	// Print the error of the attempt to execute
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	// Print all errors during execution
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	// Print whether the execution was successful
	fmt.Println("isSuccess:", result.IsSuccess())
}
```

**Result**

```bash
$ go run demo.go
========= testFunc1 =========
result: testFunc1
tryError: <nil>
execErrors: []
isSuccess: true
========= testFunc2 =========
result: <nil>
tryError: retry attempts exceeded: testFunc2
execErrors: []
isSuccess: false
```

# 特性

`Retry` 提供了一组足够满足大多数服务需求的特性。

## 1. 回调函数

`Retry` 支持回调函数。在创建重试实例时，您可以指定一个回调函数，当 `Retry` 执行特定操作时，该函数将被调用。

> [!TIP]
> 回调函数是可选的。如果您不需要回调函数，可以在创建重试实例时传递 `nil`，它将不会被调用。
>
> 您可以使用 `WithCallback` 方法来设置回调函数。

回调函数具有以下方法：

-   `OnRetry`：仅在确定发起下一次重试前调用（全部中止检查——按错误预算、总次数、retryIf——通过之后）；最后一次尝试失败导致中止时不会调用，因此 `attempts = 3` 且全部执行失败时恰好调用两次。`count` 参数表示已完成的执行次数，`delay` 参数表示即将发生的重试的退避延迟，`err` 参数表示刚失败执行的错误信息。

    ```go
    // Callback 接口用于定义重试回调函数
    // The Callback interface is used to define the retry callback function.
    type Callback interface {
    	// OnRetry 方法仅在确定发起下一次重试前调用（全部中止检查通过后）；最后一次尝试失败导致中止时不会调用。
    	// 参数：count 为已完成的执行次数，delay 为即将发生的重试的退避延迟，err 为刚失败执行的错误。
    	// The OnRetry method is called only right before the next retry is actually initiated (after all
    	// abort checks pass); it is not called when the final attempt fails and the retry aborts.
    	// Parameters: count is the number of completed executions, delay is the backoff delay of the
    	// upcoming retry, and err is the error from the execution that just failed.
    	OnRetry(count int64, delay time.Duration, err error)
    }
    ```

### 示例

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/shengyanli1982/retry"
)

// 定义一个错误变量
// Define an error variable
var err = errors.New("test") // error

// 定义一个回调结构体
// Define a callback structure
type callback struct{}

// OnRetry 方法在每次实际发起重试前被调用，接收已完成的执行次数、即将发生的重试的延迟时间和刚失败执行的错误作为参数
// The OnRetry method is called before each retry is actually initiated, receiving the number of completed executions, the delay of the upcoming retry, and the error from the execution that just failed as parameters
func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

// 定义一个可重试的函数，返回一个 nil 和一个错误
// Define a retryable function that returns a nil and an error
func testFunc() (any, error) {
	return nil, err
}

func main() {
	// 创建一个新的重试配置，并设置回调函数
	// Create a new retry configuration and set the callback function
	cfg := retry.NewConfig().WithCallback(&callback{})

	// 使用重试配置调用可重试的函数
	// Call the retryable function using the retry configuration
	result := retry.Do(testFunc, cfg)

	// 打印执行结果
	// Print the execution result
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	// Print the error of the attempt to execute
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	// Print all errors during execution
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	// Print whether the execution was successful
	fmt.Println("isSuccess:", result.IsSuccess())
}
```

**Result**

> [!NOTE]
> 下方为某一次实际运行的示例输出。`delay` 值包含随机抖动，每次运行结果不同。默认配置（`attempts = 3`）下，`OnRetry` 恰好被调用两次——每次实际发生的重试前各一次。

```bash
$ go run demo.go
OnRetry 1 1.2s test
OnRetry 2 1.9s test
result: <nil>
tryError: retry attempts exceeded: test
execErrors: []
isSuccess: false
```

# API 参考

## 退避函数

`BackoffFunc` 类型定义了退避策略函数。其 `int64` 入参为当前已完成的执行次数（`count`，从 1 开始），返回下一次重试前的退避时长 `time.Duration`。内置退避函数以 100ms 基础时间单位为倍数计算结果：

-   `FixedBackoff(interval int64) time.Duration`：返回 `interval * 100ms`。若 `interval <= 0`，返回默认延迟 `500ms`。
-   `RandomBackoff(maxInterval int64) time.Duration`：返回 `[0, maxInterval) * 100ms` 内的随机时长。若 `maxInterval <= 0`，返回默认延迟 `500ms`。
-   `ExponentialBackoff(power int64) time.Duration`：返回 `2^power * 100ms`。`power` 被钳制为最大 `36`，因为更大的幂在乘以 100ms 后会溢出 `int64`。若 `power <= 0`，返回默认延迟 `500ms`。
-   `CombineBackoffs(backoffs ...BackoffFunc) BackoffFunc`：将多个退避策略组合为一个，对同一 `count` 逐项求和。未传入任何策略时返回 `FixedBackoff`；求和结果 `<= 0` 时返回默认延迟 `500ms`。

默认退避函数为 `CombineBackoffs(ExponentialBackoff, RandomBackoff)`。

## 哨兵错误

所有哨兵中止路径都以双 `%w` 将哨兵错误与根因一起包装（`fmt.Errorf("%w: %w", sentinel, err)`），因此 `errors.Is(tryError, sentinel)` 与 `errors.Is(tryError, rootCause)` 均能命中。

| 哨兵错误 | 使用场景 |
| -------- | -------- |
| `ErrorRetryIf` | `retryIfFunc` 判定不重试时，与原始错误一起包装进 `TryError`。 |
| `ErrorRetryAttemptsExceeded` | 总 `attempts` 预算耗尽时，与最后一次错误一起包装进 `TryError`。 |
| `ErrorRetryAttemptsByErrorExceeded` | `attemptsByError` 的按错误预算耗尽时，与最后一次错误一起包装进 `TryError`。 |
| `ErrorExecErrNotFound` | 调用 `LastExecError`/`FirstExecError` 但未记录任何错误时返回（包括 `detail = false` 的情况）。 |
| `ErrorExecErrByIndexOutOfBound` | 以越界索引调用 `ExecErrorByIndex` 时返回。 |

```go
result := retry.DoWithDefault(testFunc)
if errors.Is(result.TryError(), retry.ErrorRetryAttemptsExceeded) {
	// 总重试次数耗尽；根因错误同样可以通过 errors.Is 命中
}
```

## 函数与类型

-   `Do(fn RetryableFunc, conf *Config) RetryResult`：使用给定配置执行 `fn` 并在出错时重试；配置为 `nil` 时使用默认配置。`fn` 为 `nil` 时返回真正的 `nil` 接口（而非 typed nil）。
-   `DoWithDefault(fn RetryableFunc) RetryResult`：使用默认配置执行 `fn`，等价于 `Do(fn, nil)`。`fn` 为 `nil` 时返回真正的 `nil` 接口。
-   `New(conf *Config) *Retry`：创建 `Retry` 实例。配置会被浅拷贝，归一化永不写入调用方的对象；`nil` 表示使用默认配置。
-   `(*Retry).TryOnConflict(fn RetryableFunc) *Result`：执行 `fn`，任何返回的错误都视为触发重试的冲突。`fn` 为 `nil` 时返回 `nil`。
-   `(*Retry).TryOnConflictVal(fn RetryableFunc) RetryResult`：与 `TryOnConflict` 相同，但返回 `RetryResult` 接口。`fn` 为 `nil` 时返回真正的 `nil` 接口。
-   `NewResult() *Result`：创建一个空的 `Result`，其错误列表已初始化。
-   `NewEmptyCallback() Callback`：返回 `OnRetry` 不做任何操作的回调（默认回调）。
-   `DefaultConfig() *Config`：返回一个使用默认值的新配置（与 `NewConfig` 相同）。
-   `FixConfig() *Config`：返回一个以固定 `500ms` 间隔重试的新配置（`factor = 0`、`jitter = 0`，退避函数恒返回固定值）。
-   `(*Config).WithInitDelay(delay time.Duration) *Config`：设置基础延迟（线性延迟项的乘数基数）。
-   `(*Config).WithJitter(jitter float64) *Config`：设置抖动因子。
-   `RetryableFunc = func() (any, error)`：可重试函数的类型。
-   `RetryIfFunc = func(error) bool`：重试条件函数的类型。
-   `RetryResult`：`Do`/`DoWithDefault`/`TryOnConflictVal` 返回的接口，由 `*Result` 实现。
