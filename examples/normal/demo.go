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
