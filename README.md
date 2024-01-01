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
-   `delay`: the delay time between retries. The default value is `20ms`.
-   `factor`: the retry times factor. The default value is `1.0`.
-   `retryIf`: the function to determine whether to retry. The default value is `defaultRetryIf`.
-   `backoff`: the backoff function. The default value is `defaultBackoff`.
-   `detail`: whether to record the detail errors. The default value is `false`.

Cound use following methods to set config value:

-   `WithContext`: set the context.Context object.
-   `WithCallback`: set the callback function.
-   `WithAttempts`: set the number of times to retry.
-   `WithAttemptsByErrors`: set the number of times to retry for specific error.
-   `WithDelay`: set the delay time between retries.
-   `WithFactor`: set the retry times factor.
-   `WithRetryIf`: set the function to determine whether to retry.
-   `WithBackoff`: set the backoff function.
-   `WithDetail`: set whether to record the detail errors.

### Methods

-   `Do`: retry a function call. You need to specify one config object and one function. It will return a `Result` object.
-   `DoWithDefault`: retry a function call with default config value. It will return a `Result` object.

> [!TIP]
> The `Result` object contains the result of the function call, the error of the last retry, the errors of all retries, and whether the retry is successful. If the function call fails, the default value will be returned.

### Example

Follwing is a **test.go** content.

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
