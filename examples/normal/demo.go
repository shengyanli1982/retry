package main

import (
	"fmt"

	"github.com/shengyanli1982/retry"
)

// 定义一个可重试的函数
func testFunc() (any, error) {
	// 此函数返回一个字符串 "lee" 和一个 nil 错误
	return "lee", nil
}

func main() {
	// 使用默认的重试策略调用 testFunc 函数
	result := retry.DoWithDefault(testFunc)

	// 打印执行结果
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	fmt.Println("isSuccess:", result.IsSuccess())
}
