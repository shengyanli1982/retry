package retry

import "errors"

var (
	ErrorRetryIf                      = errors.New("retry check func result is FLASE")
	ErrorRetryAttemptsExceeded        = errors.New("retry attempts exceeded")
	ErrorRetryAttemptsByErrorExceeded = errors.New("retry attempts by spec error exceeded")
)
