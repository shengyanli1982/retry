package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/shengyanli1982/retry"
)

// 定义一个错误变量
var err = errors.New("test")

// 定义一个回调结构体
type callback struct{}

// OnRetry 方法在每次实际发起重试前被调用，接收已完成的执行次数、即将发生的重试的延迟时间和刚失败执行的错误作为参数
func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

// 定义一个可重试的函数，返回一个 nil 和一个错误
func testFunc() (any, error) {
	return nil, err
}

func main() {
	// 创建一个新的重试配置，并设置回调函数
	cfg := retry.NewConfig().WithCallback(&callback{})

	// 使用重试配置调用可重试的函数
	result := retry.Do(testFunc, cfg)

	// 打印执行结果
	fmt.Println("result:", result.Data())

	// 打印尝试执行的错误
	fmt.Println("tryError:", result.TryError())

	// 打印执行过程中的所有错误
	fmt.Println("execErrors:", result.ExecErrors())

	// 打印是否成功执行
	fmt.Println("isSuccess:", result.IsSuccess())
}
