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
-   `attemptsByErrors`：特定错误的重试次数。默认值为 `map[error]uint64{}`。
-   `delay`：重试之间的延迟时间。默认值为 `200ms`。
-   `factor`：重试次数的因子。默认值为 `1.0`。
-   `retryIf`：确定是否重试的函数。默认值为 `defaultRetryIfFunc`。
-   `backoff`：退避函数。默认值为 `defaultBackoffFunc`。
-   `detail`：是否记录详细错误信息。默认值为 `false`。

您可以使用以下方法来设置配置值：

-   `WithContext`：设置上下文对象 `context.Context`。
-   `WithCallback`：设置回调函数。
-   `WithAttempts`：设置重试次数。
-   `WithAttemptsByError`：设置特定错误的重试次数。
-   `WithDelay`：设置第一次重试的延迟时间。
-   `WithFactor`：设置重试次数的因子。
-   `WithRetryIfFunc`：设置确定是否重试的函数。
-   `WithBackOffFunc`：设置退避函数。
-   `WithDetail`：设置是否记录详细错误信息。

> [!NOTE]
> 退避算法决定了重试之间的延迟时间。`Retry` 支持三种退避算法：指数退避、随机退避和固定退避。默认情况下，`Retry` 使用指数退避与随机退避值之和。
>
> 您可以使用 `WithBackOffFunc` 方法来设置退避算法。
>
> **eg**: backoff = backoffFunc(factor \* count + jitter \* rand.Float64()) \* 100 \* Millisecond + delay

### 方法

-   `Do`: 通过指定配置对象和函数来重试函数调用。它返回一个 `Result` 对象。
-   `DoWithDefault`: 使用默认配置值来重试函数调用。它返回一个 `Result` 对象。

> [!TIP]
> 在 `Result` 对象内包含函数调用的结果、最后一次重试的错误、所有重试的错误以及重试是否成功。如果函数调用失败，将返回默认值。

### 执行结果

在重试之后，`Retry` 返回一个 `Result` 对象。`Result` 对象提供以下方法：

-   `Data`: 获取成功调用函数的结果。类型为 `interface{}`。
-   `TryError`: 获取重试操作的错误。如果重试成功，则值为 `nil`。
-   `ExecErrors`: 获取所有重试的错误。
-   `IsSuccess`: 检查重试操作是否成功。
-   `LastExecError`: 获取最后一次重试的错误。
-   `FirstExecError`: 获取第一次重试的错误。
-   `ExecErrorByIndex`: 通过索引获取特定重试的错误。

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
tryError: retry attempts exceeded
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

-   `OnRetry`：在重试时调用。`count` 参数表示当前重试次数，`delay` 参数表示下一次重试的延迟时间，`err` 参数表示上一次重试的错误信息。

    ```go
    // Callback 接口用于定义重试回调函数
    // The Callback interface is used to define the retry callback function.
    type Callback interface {
    	// OnRetry 方法在每次重试时调用，传入当前的重试次数、延迟时间和错误信息
    	// The OnRetry method is called on each retry, passing in the current retry count, delay time, and error information
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

// OnRetry 方法在每次重试时被调用，接收重试次数、延迟时间和错误作为参数
// The OnRetry method is called each time a retry is performed, receiving the number of retries, delay time, and error as parameters
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

```bash
$ go run demo.go
OnRetry 1 1s test
OnRetry 2 1.5s test
OnRetry 3 2.4s test
result: <nil>
tryError: retry attempts exceeded
execErrors: []
isSuccess: false
```
