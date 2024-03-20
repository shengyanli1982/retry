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
