package retry

import "time"

// Callback 接口用于定义重试回调函数
type Callback interface {
	// OnRetry 方法仅在确定发起下一次重试前调用（全部中止检查通过后）；最后一次尝试失败导致中止时不会调用。
	// 参数：count 为已完成的执行次数，delay 为即将发生的重试的退避延迟，err 为刚失败执行的错误。
	OnRetry(count int64, delay time.Duration, err error)
}

// RetryResult 接口定义了执行结果的相关方法，由 *Result 实现，是 Do/DoWithDefault/TryOnConflictVal 的返回类型
type RetryResult interface {
	// Data 方法返回执行结果的数据
	Data() any

	// TryError 方法返回尝试执行时的错误
	TryError() error

	// ExecErrors 方法返回所有执行错误的列表（仅 Config.detail 为 true 时记录，否则为空切片）
	ExecErrors() []error

	// IsSuccess 方法返回执行是否成功
	IsSuccess() bool

	// LastExecError 方法返回最后一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
	LastExecError() error

	// FirstExecError 方法返回第一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
	FirstExecError() error

	// ExecErrorByIndex 方法返回指定索引处的执行错误；索引越界时返回 ErrorExecErrByIndexOutOfBound
	ExecErrorByIndex(idx int) error

	// Count 方法返回执行的次数
	Count() int64
}
