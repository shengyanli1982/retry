package retry

import "time"

// Callback 接口用于定义重试回调函数
// The Callback interface is used to define the retry callback function.
type Callback interface {
	// OnRetry 方法在每次重试时调用，传入当前的重试次数、延迟时间和错误信息
	// The OnRetry method is called on each retry, passing in the current retry count, delay time, and error information
	OnRetry(count int64, delay time.Duration, err error)
}

// RetryResult 接口定义了执行结果的相关方法
// The RetryResult interface defines methods related to execution results
type RetryResult = interface {
	// Data 方法返回执行结果的数据
	// The Data method returns the data of the execution result
	Data() any

	// TryError 方法返回尝试执行时的错误
	// The TryError method returns the error when trying to execute
	TryError() error

	// ExecErrors 方法返回所有执行错误的列表
	// The ExecErrors method returns a list of all execution errors
	ExecErrors() []error

	// IsSuccess 方法返回执行是否成功
	// The IsSuccess method returns whether the execution was successful
	IsSuccess() bool

	// LastExecError 方法返回最后一次执行的错误
	// The LastExecError method returns the error of the last execution
	LastExecError() error

	// FirstExecError 方法返回第一次执行的错误
	// The FirstExecError method returns the error of the first execution
	FirstExecError() error

	// ExecErrorByIndex 方法返回指定索引处的执行错误
	// The ExecErrorByIndex method returns the execution error at the specified index
	ExecErrorByIndex(idx int) error

	// Count 方法返回执行的次数
	// The Count method returns the number of executions
	Count() int64
}
