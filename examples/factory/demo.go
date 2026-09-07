package main

import (
	"errors"
	"fmt"

	"github.com/shengyanli1982/retry"
)

// 定义一个可重试的函数 testFunc1
func testFunc1() (any, error) {
	// 此函数返回一个字符串 "testFunc1" 和一个 nil 错误
	return "testFunc1", nil
}

// 定义一个可重试的函数 testFunc2
func testFunc2() (any, error) {
	// 此函数返回一个 nil 和一个新的错误 "testFunc2"
	return nil, errors.New("testFunc2")
}

func main() {
	// 使用默认的配置创建一个新的重试实例
	r := retry.New(nil)

	// 尝试执行 testFunc1 函数，如果遇到冲突则进行重试
	result := r.TryOnConflict(testFunc1)

	// 打印 testFunc1 执行结果
	fmt.Println("========= testFunc1 =========")

	// 打印执行结果
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	fmt.Println("isSuccess:", result.IsSuccess())

	// 尝试执行 testFunc2 函数，如果遇到冲突则进行重试
	result = r.TryOnConflict(testFunc2)

	// 打印 testFunc2 执行结果
	fmt.Println("========= testFunc2 =========")

	// 打印执行结果
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	fmt.Println("isSuccess:", result.IsSuccess())
}
