package retry

import "errors"

var (
	// ErrorRetryIf 表示重试检查函数的结果为FALSE的错误。
	// retryIfFunc 判定不重试时，TryError 以 fmt.Errorf("%w: %w", ErrorRetryIf, err) 同时包装本哨兵与根因，errors.Is 对两者均命中。
	ErrorRetryIf = errors.New("retry check func result is FALSE")

	// ErrorRetryAttemptsExceeded 表示重试次数超过限制的错误。
	// 总 attempts 耗尽时，TryError 以双 %w 同时包装本哨兵与最后一次执行的错误（消息形如 "retry attempts exceeded: <根因>"）。
	ErrorRetryAttemptsExceeded = errors.New("retry attempts exceeded")

	// ErrorRetryAttemptsByErrorExceeded 表示由于特定错误导致的重试次数超过限制的错误。
	// attemptsByError 预算耗尽时，TryError 以双 %w 同时包装本哨兵与最后一次执行的错误（消息形如 "retry attempts by spec error exceeded: <根因>"）。
	ErrorRetryAttemptsByErrorExceeded = errors.New("retry attempts by spec error exceeded")

	// ErrorExecErrByIndexOutOfBound 表示由于索引越界导致的执行错误。
	// ExecErrorByIndex 在索引超出已记录错误列表范围时返回本哨兵。
	ErrorExecErrByIndexOutOfBound = errors.New("exec error by index out of bound")

	// ErrorExecErrNotFound 表示未找到执行错误。
	// LastExecError/FirstExecError 在错误列表为空时返回本哨兵（包括 detail=false 未记录任何错误的情况）。
	ErrorExecErrNotFound = errors.New("exec error not found")
)

// wrappedError 是 fmt.Errorf("%w: %w", sentinel, cause) 的零分配替代品。
// 实现 Go 1.20+ 的 Unwrap() []error 接口，使 errors.Is 同时匹配 sentinel 和 cause。
type wrappedError struct {
	sentinel error
	cause    error
}

// Error 返回错误消息，格式为 "sentinel: cause"
func (e wrappedError) Error() string {
	return e.sentinel.Error() + ": " + e.cause.Error()
}

// Unwrap 返回包装的两个错误，供 errors.Is/errors.As 遍历
func (e wrappedError) Unwrap() []error {
	return []error{e.sentinel, e.cause}
}
