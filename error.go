package retry

import "errors"

var (
	// ErrorRetryIf 表示重试检查函数的结果为FALSE的错误
	// ErrorRetryIf represents an error when the retry check function result is FALSE
	ErrorRetryIf = errors.New("retry check func result is FALSE")

	// ErrorRetryAttemptsExceeded 表示重试次数超过限制的错误
	// ErrorRetryAttemptsExceeded represents an error when the retry attempts exceeded the limit
	ErrorRetryAttemptsExceeded = errors.New("retry attempts exceeded")

	// ErrorRetryAttemptsByErrorExceeded 表示由于特定错误导致的重试次数超过限制的错误
	// ErrorRetryAttemptsByErrorExceeded represents an error when the retry attempts exceeded the limit due to a specific error
	ErrorRetryAttemptsByErrorExceeded = errors.New("retry attempts by spec error exceeded")

	// ErrorExecErrByIndexOutOfBound 表示由于索引越界导致的执行错误
	// ErrorExecErrByIndexOutOfBound represents an execution error caused by index out of bound
	ErrorExecErrByIndexOutOfBound = errors.New("exec error by index out of bound")

	// ErrorExecErrNotFound 表示未找到执行错误
	// ErrorExecErrNotFound represents an error when the execution error is not found
	ErrorExecErrNotFound = errors.New("exec error not found")
)
