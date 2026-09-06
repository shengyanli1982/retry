English | [中文](./README_CN.md)

<div align="center">
	<h1>Retry</h1>
    <p>A simple, dependency-free module for effortless function retrying in various scenarios.</p>
	<img src="assets/logo.png" alt="logo" width="350px">
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/shengyanli1982/retry)](https://goreportcard.com/report/github.com/shengyanli1982/retry)
[![Build Status](https://github.com/shengyanli1982/retry/actions/workflows/test.yaml/badge.svg)](github.com/shengyanli1982/retry/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shengyanli1982/retry.svg)](https://pkg.go.dev/github.com/shengyanli1982/retry)

# Introduction

`Retry` is a lightweight module for retrying function calls. It is simple, easy to use, and has no third-party dependencies. It is designed for scenarios where you need to retry a function call.

`Retry` provides the following features:

1. Retry a function call a specified number of times.
2. Retry a function call a specified number of times for specific errors.
3. Support action callback functions.
4. Support jitter factor for delay.
5. Support exponential backoff delay, random delay, and fixed delay.
6. Support recording detailed errors for each failed retry.

# Advantages

-   Simple and user-friendly
-   No external dependencies required
-   Efficient memory usage
-   Supports callback functions

# Installation

```bash
go get github.com/shengyanli1982/retry
```

# Quick Start

Using `Retry` is simple. Just one line of code is needed to retry a function call.

## 1. Normal Model

### Config

`Retry` provides a config object to customize the retry behavior. The config object has the following fields:

-   `ctx`: The context.Context object. The default value is `context.Background()`.
-   `callback`: The callback function. The default value is `&emptyCallback{}`.
-   `attempts`: The number of retry attempts. The default value is `3`.
-   `attemptsByError`: The number of retry attempts for specific errors. The default value is `map[error]uint64{}`.
-   `delay`: The base delay used as the multiplier of the linear delay component for each retry (see the backoff formula below). The default value is `500ms`.
-   `factor`: The retry times factor. The default value is `1.0`.
-   `jitter`: The jitter factor that adds randomness to the linear delay component. The default value is `3.0`.
-   `retryIfFunc`: The function to determine whether to retry. The default value is `defaultRetryIfFunc`.
-   `backoffFunc`: The backoff function. The default value is `defaultBackoffFunc`.
-   `detail`: Whether to record detailed errors. The default value is `false`.

You can use the following methods to set config values:

-   `WithContext`: Set the context.Context object.
-   `WithCallback`: Set the callback function.
-   `WithAttempts`: Set the number of retry attempts. The value is normalized when passed to `New`: `0` falls back to the default `3`, and values `>= 65535` are clamped to `65534`.
-   `WithAttemptsByError`: Set the number of retry attempts for specific errors.
-   `WithInitDelay`: Set the base delay (the multiplier base of the linear delay component for each retry).
-   `WithJitter`: Set the jitter factor.
-   `WithFactor`: Set the retry times factor.
-   `WithRetryIfFunc`: Set the function to determine whether to retry.
-   `WithBackOffFunc`: Set the backoff function.
-   `WithDetail`: Set whether to record detailed errors.

> [!NOTE]
> The backoff algorithm determines the delay time between retries. `Retry` supports three backoff algorithms: exponential backoff, random backoff, and fixed backoff. By default, `Retry` uses exponential backoff with random backoff values added to the delay time.
>
> You can use the `WithBackOffFunc` method to set the backoff algorithm.
>
> **eg**: backoff = backoffFunc(count) + trunc(jitter \* rand.Float64() + factor \* count) \* delay
>
> Where `count` is the number of completed executions (starting from 1), `rand.Float64()` returns a value in `[0, 1)`, and `trunc` is the float64-to-`time.Duration` conversion, which truncates toward zero — so the linear component is quantized to an integer multiple of `delay`. The backoff function receives `count` as its only parameter; the built-in backoff functions compute their results in units of a 100ms base interval (see [API Reference](#api-reference)).

### Methods

-   `Do`: Retry a function call by specifying a config object and a function. It returns a `RetryResult` interface value (implemented by `*Result`).
-   `DoWithDefault`: Retry a function call with default config values. It returns a `RetryResult` interface value (implemented by `*Result`).

> [!TIP]
> The returned `RetryResult` value contains the result of the function call, the error of the last retry, the errors of all retries, and whether the retry was successful. If the function call fails, the default value will be returned.

### Exec Result

After retrying, `Retry` returns a `RetryResult` interface value (implemented by `*Result`). The `RetryResult` interface provides the following methods:

-   `Data`: Get the result of the successfully called function. The type is `interface{}`.
-   `TryError`: Get the error of the retry action. If the retry is successful, the value is `nil`.
-   `ExecErrors`: Get the errors of all retries. Errors are recorded only when `detail` is `true`; otherwise the list stays empty.
-   `IsSuccess`: Check if the retry action was successful.
-   `LastExecError`: Get the last error of the retries. Returns the `ErrorExecErrNotFound` sentinel when no errors were recorded.
-   `FirstExecError`: Get the first error of the retries. Returns the `ErrorExecErrNotFound` sentinel when no errors were recorded.
-   `ExecErrorByIndex`: Get the error of a specific retry by index. Returns the `ErrorExecErrByIndexOutOfBound` sentinel when the index is out of range.
-   `Count`: Get the number of executions performed. The type is `int64`.

### Example

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

## 2. Factory Model

The Factory Model provides all the same retry functions and features as the Normal Model. It uses the same `Config`, `Methods`, `Result`, and `Callback`.

The only difference is that the `Retry` object is created using the `New` method. Then you can use the `TryOnConflict` method to retry the function call with the same parameters.

### Example

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

# Features

`Retry` provides a set of features that are sufficient for most services.

## 1. Callback

`Retry` supports callback functions. You can specify a callback function when creating a retry, and it will be called when the `Retry` performs certain actions.

> [!TIP]
> Callback functions are optional. If you don't need a callback function, you can pass `nil` when creating a retry, and it won't be called.
>
> You can use the `WithCallback` method to set a callback function.

The callback function has the following methods:

-   `OnRetry`: called only right before a retry is actually initiated, after all abort checks (per-error budget, total attempts, retryIf) have passed — it is not called when the final attempt fails and the retry aborts, so with `attempts = 3` and every execution failing it is called exactly twice. The `count` parameter represents the number of completed executions, the `delay` parameter represents the backoff delay of the upcoming retry, and the `err` parameter represents the error from the execution that just failed.

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

### Example

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
> Sample output from one actual run. The `delay` values contain random jitter and differ between runs. With the default config (`attempts = 3`), `OnRetry` is called exactly twice — once before each of the two actual retries.

```bash
$ go run demo.go
OnRetry 1 1.2s test
OnRetry 2 1.9s test
result: <nil>
tryError: retry attempts exceeded: test
execErrors: []
isSuccess: false
```

# API Reference

## Backoff Functions

The `BackoffFunc` type defines a backoff strategy function. Its `int64` parameter is the current number of completed executions (`count`, starting from 1), and it returns the backoff `time.Duration` before the next retry. The built-in backoff functions compute their results in units of a 100ms base interval:

-   `FixedBackoff(interval int64) time.Duration`: Returns `interval * 100ms`. If `interval <= 0`, returns the default delay `500ms`.
-   `RandomBackoff(maxInterval int64) time.Duration`: Returns a random duration in `[0, maxInterval) * 100ms`. If `maxInterval <= 0`, returns the default delay `500ms`.
-   `ExponentialBackoff(power int64) time.Duration`: Returns `2^power * 100ms`. `power` is clamped to a maximum of `36`, because larger powers would overflow `int64` after multiplying by 100ms. If `power <= 0`, returns the default delay `500ms`.
-   `CombineBackoffs(backoffs ...BackoffFunc) BackoffFunc`: Combines multiple backoff strategies into one by summing their results for the same `count`. If no strategy is given, returns `FixedBackoff`. If the sum is `<= 0`, returns the default delay `500ms`.

The default backoff function is `CombineBackoffs(ExponentialBackoff, RandomBackoff)`.

## Sentinel Errors

Every sentinel abort path wraps the sentinel error together with the root cause using double `%w` (`fmt.Errorf("%w: %w", sentinel, err)`), so both `errors.Is(tryError, sentinel)` and `errors.Is(tryError, rootCause)` match.

| Sentinel | Used when |
| -------- | --------- |
| `ErrorRetryIf` | `retryIfFunc` decides not to retry; wrapped with the original error in `TryError`. |
| `ErrorRetryAttemptsExceeded` | The total `attempts` budget is exhausted; wrapped with the last error in `TryError`. |
| `ErrorRetryAttemptsByErrorExceeded` | The per-error budget from `attemptsByError` is exhausted; wrapped with the last error in `TryError`. |
| `ErrorExecErrNotFound` | `LastExecError`/`FirstExecError` is called but no errors were recorded (including `detail = false`). |
| `ErrorExecErrByIndexOutOfBound` | `ExecErrorByIndex` is called with an out-of-range index. |

```go
result := retry.DoWithDefault(testFunc)
if errors.Is(result.TryError(), retry.ErrorRetryAttemptsExceeded) {
	// total attempts exhausted; the root cause is also reachable via errors.Is
}
```

## Functions and Types

-   `Do(fn RetryableFunc, conf *Config) RetryResult`: Executes `fn` with the given config and retries on error; a `nil` config means defaults. Returns a true `nil` interface (not a typed nil) when `fn` is `nil`.
-   `DoWithDefault(fn RetryableFunc) RetryResult`: Executes `fn` with the default config, equivalent to `Do(fn, nil)`. Returns a true `nil` interface when `fn` is `nil`.
-   `New(conf *Config) *Retry`: Creates a `Retry` instance. The config is shallow-copied, and normalization never writes the caller's object; `nil` means defaults.
-   `(*Retry).TryOnConflict(fn RetryableFunc) *Result`: Runs `fn` and treats any returned error as a conflict that triggers a retry. Returns `nil` when `fn` is `nil`.
-   `(*Retry).TryOnConflictVal(fn RetryableFunc) RetryResult`: Same as `TryOnConflict` but returns the `RetryResult` interface. Returns a true `nil` interface when `fn` is `nil`.
-   `NewResult() *Result`: Creates an empty `Result` with an initialized error list.
-   `NewEmptyCallback() Callback`: Returns a callback whose `OnRetry` does nothing (the default callback).
-   `DefaultConfig() *Config`: Returns a new config with default values (same as `NewConfig`).
-   `FixConfig() *Config`: Returns a new config that retries at a fixed `500ms` interval (`factor = 0`, `jitter = 0`, and a constant backoff function).
-   `(*Config).WithInitDelay(delay time.Duration) *Config`: Sets the base delay (the multiplier base of the linear delay component).
-   `(*Config).WithJitter(jitter float64) *Config`: Sets the jitter factor.
-   `RetryableFunc = func() (any, error)`: The type of a retryable function.
-   `RetryIfFunc = func(error) bool`: The type of the retry-condition function.
-   `RetryResult`: The interface returned by `Do`/`DoWithDefault`/`TryOnConflictVal`, implemented by `*Result`.
