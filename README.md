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
-   `attemptsByErrors`: The number of retry attempts for specific errors. The default value is `map[error]uint64{}`.
-   `delay`: The delay time between retries. The default value is `200ms`.
-   `factor`: The retry times factor. The default value is `1.0`.
-   `retryIf`: The function to determine whether to retry. The default value is `defaultRetryIfFunc`.
-   `backoff`: The backoff function. The default value is `defaultBackoffFunc`.
-   `detail`: Whether to record detailed errors. The default value is `false`.

You can use the following methods to set config values:

-   `WithContext`: Set the context.Context object.
-   `WithCallback`: Set the callback function.
-   `WithAttempts`: Set the number of retry attempts.
-   `WithAttemptsByError`: Set the number of retry attempts for specific errors.
-   `WithDelay`: Set the delay time for the first retry.
-   `WithFactor`: Set the retry times factor.
-   `WithRetryIfFunc`: Set the function to determine whether to retry.
-   `WithBackOffFunc`: Set the backoff function.
-   `WithDetail`: Set whether to record detailed errors.

> [!NOTE]
> The backoff algorithm determines the delay time between retries. `Retry` supports three backoff algorithms: exponential backoff, random backoff, and fixed backoff. By default, `Retry` uses exponential backoff with random backoff values added to the delay time.
>
> You can use the `WithBackOffFunc` method to set the backoff algorithm.
>
> **eg**: backoff = backoffFunc(factor \* count + jitter \* rand.Float64()) \* 100 \* Millisecond + delay

### Methods

-   `Do`: Retry a function call by specifying a config object and a function. It returns a `Result` object.
-   `DoWithDefault`: Retry a function call with default config values. It returns a `Result` object.

> [!TIP]
> The `Result` object contains the result of the function call, the error of the last retry, the errors of all retries, and whether the retry was successful. If the function call fails, the default value will be returned.

### Exec Result

After retrying, `Retry` returns a `Result` object. The `Result` object provides the following methods:

-   `Data`: Get the result of the successfully called function. The type is `interface{}`.
-   `TryError`: Get the error of the retry action. If the retry is successful, the value is `nil`.
-   `ExecErrors`: Get the errors of all retries.
-   `IsSuccess`: Check if the retry action was successful.
-   `LastExecError`: Get the last error of the retries.
-   `FirstExecError`: Get the first error of the retries.
-   `ExecErrorByIndex`: Get the error of a specific retry by index.

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
tryError: retry attempts exceeded
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

-   `OnRetry`: called when retrying. The `count` parameter represents the current retry count, the `delay` parameter represents the delay time for the next retry, and the `err` parameter represents the error from the last retry.

    ```go
    // Callback 接口用于定义重试回调函数
    // The Callback interface is used to define the retry callback function.
    type Callback interface {
    	// OnRetry 方法在每次重试时调用，传入当前的重试次数、延迟时间和错误信息
    	// The OnRetry method is called on each retry, passing in the current retry count, delay time, and error information
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
