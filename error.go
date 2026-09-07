package retry

import "errors"

var (
	// ErrorRetryIf 表示重试检查函数的结果为FALSE的错误。
	// retryIfFunc 判定不重试时，TryError 以 fmt.Errorf("%w: %w", ErrorRetryIf, err) 同时包装本哨兵与根因，errors.Is 对两者均命中。
	// ErrorRetryIf represents an error when the retry check function result is FALSE.
	// When retryIfFunc decides not to retry, TryError wraps both this sentinel and the root cause via
	// fmt.Errorf("%w: %w", ErrorRetryIf, err), so errors.Is matches either.
	ErrorRetryIf = errors.New("retry check func result is FALSE")

	// ErrorRetryAttemptsExceeded 表示重试次数超过限制的错误。
	// 总 attempts 耗尽时，TryError 以双 %w 同时包装本哨兵与最后一次执行的错误（消息形如 "retry attempts exceeded: <根因>"）。
	// ErrorRetryAttemptsExceeded represents an error when the retry attempts exceeded the limit.
	// When the total attempts budget is exhausted, TryError wraps both this sentinel and the last execution
	// error with double %w (message like "retry attempts exceeded: <root cause>").
	ErrorRetryAttemptsExceeded = errors.New("retry attempts exceeded")

	// ErrorRetryAttemptsByErrorExceeded 表示由于特定错误导致的重试次数超过限制的错误。
	// attemptsByError 预算耗尽时，TryError 以双 %w 同时包装本哨兵与最后一次执行的错误（消息形如 "retry attempts by spec error exceeded: <根因>"）。
	// ErrorRetryAttemptsByErrorExceeded represents an error when the retry attempts exceeded the limit due to a specific error.
	// When the attemptsByError budget is exhausted, TryError wraps both this sentinel and the last execution
	// error with double %w (message like "retry attempts by spec error exceeded: <root cause>").
	ErrorRetryAttemptsByErrorExceeded = errors.New("retry attempts by spec error exceeded")

	// ErrorExecErrByIndexOutOfBound 表示由于索引越界导致的执行错误。
	// ExecErrorByIndex 在索引超出已记录错误列表范围时返回本哨兵。
	// ErrorExecErrByIndexOutOfBound represents an execution error caused by index out of bound.
	// ExecErrorByIndex returns this sentinel when the index is out of range of the recorded error list.
	ErrorExecErrByIndexOutOfBound = errors.New("exec error by index out of bound")

	// ErrorExecErrNotFound 表示未找到执行错误。
	// LastExecError/FirstExecError 在错误列表为空时返回本哨兵（包括 detail=false 未记录任何错误的情况）。
	// ErrorExecErrNotFound represents an error when the execution error is not found.
	// LastExecError/FirstExecError return this sentinel when the error list is empty (including when
	// detail=false recorded no errors).
	ErrorExecErrNotFound = errors.New("exec error not found")
)

// wrappedError 是 fmt.Errorf("%w: %w", sentinel, cause) 的零分配替代品。
// 实现 Go 1.20+ 的 Unwrap() []error 接口，使 errors.Is 同时匹配 sentinel 和 cause。
// wrappedError is a zero-allocation replacement for fmt.Errorf("%w: %w", sentinel, cause).
// It implements the Go 1.20+ Unwrap() []error interface, allowing errors.Is to match both sentinel and cause.
type wrappedError struct {
	sentinel error
	cause    error
}

// Error 返回错误消息，格式为 "sentinel: cause"
// Error returns the error message in the format "sentinel: cause"
func (e wrappedError) Error() string {
	return e.sentinel.Error() + ": " + e.cause.Error()
}

// Unwrap 返回包装的两个错误，供 errors.Is/errors.As 遍历
// Unwrap returns the two wrapped errors for errors.Is/errors.As traversal
func (e wrappedError) Unwrap() []error {
	return []error{e.sentinel, e.cause}
}
