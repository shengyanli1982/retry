package retry

import "errors"

var (
	ErrorRetryIf                      = errors.New("retry check func result is FALSE")
	ErrorRetryAttemptsExceeded        = errors.New("retry attempts exceeded")
	ErrorRetryAttemptsByErrorExceeded = errors.New("retry attempts by spec error exceeded")
	ErrorExecErrByIndexOutOfBound     = errors.New("exec error by index out of bound")
	ErrorExecErrNotFound              = errors.New("exec error not found")
)
