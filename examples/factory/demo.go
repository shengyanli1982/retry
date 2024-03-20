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
