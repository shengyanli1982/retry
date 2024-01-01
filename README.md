<div align="center">
	<h1>Retry</h1>
	<img src="assets/logo.png" alt="logo" width="300px">
    <h4>A lightweight function retrying module</h4>
</div>

# Introduction

`Retry` is a lightweight function retrying module. It is simple and easy to use, and has no third-party dependencies. It is suitable for scenarios where you need to retry a function call.

`Retry` is very simple, it only has one function `Do` or `DoWithDefault`, which can be used to retry a function call.

`Retry` supports the following features:

1. specified number of times for retrying
2. specified number of times for specific error
3. support action callback functions
4. support jitter factor for delay
5. support exponential backoff delay, random delay and fix delay.
6. support detail errors which every retry failed.

# Advantage

-   Simple and easy to use
-   No third-party dependencies
-   Low memory usage
-   Support action callback functions

# Installation

```bash
go get github.com/shengyanli1982/retry
```

# Quick Start

`Retry` is very simple to use. Just one line of code can be used to retry a function call.

### Config

`Retry` has a config object, which can be used to configure the retry behavior. The config object has the following fields:

-   `ctx`: the context.Context object. The default value is `context.Background()`.
-   `cb`: the callback function. The default value is `&emptyCallback{}`.
-   `attempts`: the number of times to retry. The default value is `3`.
-   `attemptsByErrors`: the number of times to retry for specific error. The default value is `map[error]uint64{}`.
-   `delay`: the delay time between retries. The default value is `200ms`.
-   `factor`: the retry times factor. The default value is `1.0`.
-   `retryIf`: the function to determine whether to retry. The default value is `defaultRetryIf`.
-   `backoff`: the backoff function. The default value is `defaultBackoff`.
-   `detail`: whether to record the detail errors. The default value is `false`.

Cound use following methods to set config value:

-   `WithContext`: set the context.Context object.
-   `WithCallback`: set the callback function.
-   `WithAttempts`: set the number of times to retry.
-   `WithAttemptsByErrors`: set the number of times to retry for specific error.
-   `WithDelay`: set the delay time for frist retry.
-   `WithFactor`: set the retry times factor.
-   `WithRetryIf`: set the function to determine whether to retry.
-   `WithBackoff`: set the backoff function.
-   `WithDetail`: set whether to record the detail errors.

> [!NOTE]
> Backoff algorithm is used to calculate the delay time between retries. `Retry` supports three backoff algorithms: exponential backoff, random backoff and fix backoff. The default backoff algorithm is exponential backoff and random backoff.
>
> You can use `WithBackoff` method to set backoff algorithm.
>
> **eg**: backoff = BackOffFunc(factor \* count + jitter \* rand.Float64()) + delay

### Methods

-   `Do`: retry a function call. You need to specify one config object and one function. It will return a `Result` object.
-   `DoWithDefault`: retry a function call with default config value. It will return a `Result` object.

> [!TIP]
> The `Result` object contains the result of the function call, the error of the last retry, the errors of all retries, and whether the retry is successful. If the function call fails, the default value will be returned.

### Example

```go
package main

import (
	"fmt"

	"github.com/shengyanli1982/retry"
)

// retryable function
func testFunc() (any, error) {
	return "lee", nil
}

func main() {
	// retry call
	result := retry.DoWithDefault(testFunc)

	// result
	fmt.Println("result:", result.Data())
	fmt.Println("tryError:", result.TryError())
	fmt.Println("execErrors:", result.ExecErrors())
	fmt.Println("isSuccess:", result.IsSuccess())
}
```

**Result**

```bash
$ go run test.go
result: lee
tryError: <nil>
execErrors: []
isSuccess: true
```

# Features

`Retry` provides features not many but enough for most services.

## 1. Callback

`Retry` supports action callback function. Specify a callback functions when create a retry, and the callback function will be called when the `Retry` do some actions.

> [!TIP]
> Callback functions is not required that you can use `Retry` without callback functions. Set `nil` when create a retry, and the callback function will not be called.
>
> You can use `WithCallback` method to set callback functions.

The callback function has the following methods:

-   `OnRetry` : called when retrying. `count` is the current retry count, `delay` is the delay time for next time, `err` is the error of the last retry.

    ```go
    type Callback interface {
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

var e = errors.New("test") // error

type callback struct{}

// OnRetry is called when retrying
func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

// retryable function
func testFunc() (any, error) {
	return nil, e
}

func main() {
	cfg := retry.NewConfig().WithCallback(&callback{})

	// retry call
	result := retry.Do(testFunc, cfg)

	// result
	fmt.Println("result:", result.Data())
	fmt.Println("tryError:", result.TryError())
	fmt.Println("execErrors:", result.ExecErrors())
	fmt.Println("isSuccess:", result.IsSuccess())
}
```

**Result**

```go
$ go run test.go
OnRetry 1 1s test
OnRetry 2 2.2s test
OnRetry 3 2.4s test
result: <nil>
tryError: retry attempts exceeded
execErrors: []
isSuccess: false
```
