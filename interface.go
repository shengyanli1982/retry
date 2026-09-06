package retry

import "time"

// Callback 接口用于定义重试回调函数
// The Callback interface is used to define the retry callback function.
type Callback interface {
	// OnRetry 方法仅在确定发起下一次重试前调用（全部中止检查通过后）；最后一次尝试失败导致中止时不会调用。
	// 参数：count 为已完成的执行次数，delay 为即将发生的重试的退避延迟，err 为刚失败执行的错误。
	// The OnRetry method is called only right before the next retry is actually initiated (after all
	// abort checks pass); it is not called when the final attempt fails and the retry aborts.
	// Parameters: count is the number of completed executions, delay is the backoff delay of the
	// upcoming retry, and err is the error from the execution that just failed.
	OnRetry(count int64, delay time.Duration, err error)
}

// RetryResult 接口定义了执行结果的相关方法，由 *Result 实现，是 Do/DoWithDefault/TryOnConflictVal 的返回类型
// The RetryResult interface defines methods related to execution results. It is implemented by
// *Result and is the return type of Do/DoWithDefault/TryOnConflictVal.
type RetryResult interface {
	// Data 方法返回执行结果的数据
	// The Data method returns the data of the execution result
	Data() any

	// TryError 方法返回尝试执行时的错误
	// The TryError method returns the error when trying to execute
	TryError() error

	// ExecErrors 方法返回所有执行错误的列表（仅 Config.detail 为 true 时记录，否则为空切片）
	// The ExecErrors method returns a list of all execution errors (recorded only when Config.detail
	// is true; otherwise an empty slice)
	ExecErrors() []error

	// IsSuccess 方法返回执行是否成功
	// The IsSuccess method returns whether the execution was successful
	IsSuccess() bool

	// LastExecError 方法返回最后一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
	// The LastExecError method returns the error of the last execution; it returns ErrorExecErrNotFound
	// when the error list is empty (including when detail=false recorded nothing)
	LastExecError() error

	// FirstExecError 方法返回第一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
	// The FirstExecError method returns the error of the first execution; it returns ErrorExecErrNotFound
	// when the error list is empty (including when detail=false recorded nothing)
	FirstExecError() error

	// ExecErrorByIndex 方法返回指定索引处的执行错误；索引越界时返回 ErrorExecErrByIndexOutOfBound
	// The ExecErrorByIndex method returns the execution error at the specified index; it returns
	// ErrorExecErrByIndexOutOfBound when the index is out of range
	ExecErrorByIndex(idx int) error

	// Count 方法返回执行的次数
	// The Count method returns the number of executions
	Count() int64
}
