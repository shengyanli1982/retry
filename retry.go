package retry

import (
	"math/rand"
	"time"
)

// data 为执行结果，tryError 为尝试执行时的错误，execErrors 为执行过程中的错误
// data is the result of the execution, tryError is the error when trying to execute, and execErrors is the error during the execution.
type Result struct {
	count      uint64
	data       any
	tryError   error
	execErrors []error
}

// NewResult 方法用于创建一个新的执行结果
// The NewResult method is used to create a new execution result.
func NewResult() *Result {
	return &Result{execErrors: make([]error, 0)}
}

// Data 方法用于获取执行结果
// The Data method is used to get the execution result.
func (r *Result) Data() any {
	return r.data
}

// TryError 方法用于获取尝试执行时的错误
// The TryError method is used to get the error when trying to execute.
func (r *Result) TryError() error {
	return r.tryError
}

// ExecErrors 方法用于获取执行过程中的错误
// The ExecErrors method is used to get the error during the execution.
func (r *Result) ExecErrors() []error {
	return r.execErrors
}

// IsSuccess 方法用于判断执行结果是否成功
// The IsSuccess method is used to determine whether the execution result is successful.
func (r *Result) IsSuccess() bool {
	return r.tryError == nil
}

// LastExecError 方法用于获取最后一个执行过程中的错误
// The LastExecError method is used to get the last error during the execution.
func (r *Result) LastExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[len(r.execErrors)-1]
	}
	return ErrorExecErrNotFound
}

// FirstExecError 方法用于获取第一个执行过程中的错误
// The FirstExecError method is used to get the first error during the execution.
func (r *Result) FirstExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[0]
	}
	return ErrorExecErrNotFound
}

// ExecErrorByIndex 方法用于获取指定索引的执行过程中的错误
// The ExecErrorByIndex method is used to get the error during the execution at the specified index.
func (r *Result) ExecErrorByIndex(n int) error {
	if len(r.execErrors) >= 0 && n < len(r.execErrors) {
		return r.execErrors[n]
	}
	return ErrorExecErrByIndexOutOfBound
}

// Count 方法用于获取执行次数
// The Count method is used to get the number of executions.
func (r *Result) Count() int64 {
	return int64(r.count)
}

// RetryableFunc 方法用于定义待执行的函数
// The RetryableFunc method is used to define the function to be executed.
type RetryableFunc = func() (any, error)

// config 为重试配置
// config is the Retry configuration
type Retry struct {
	config *Config
}

// New 方法用于创建一个新的重试实例
// The New method is used to create a new retry instance.
func New(conf *Config) *Retry {
	conf = isConfigValid(conf)
	return &Retry{config: conf}
}

// TryOnConflict 方法用于执行重试
// The TryOnConflict method is used to execute the retry.
func (r *Retry) TryOnConflict(fn RetryableFunc) *Result {
	// fn 为 nil 时直接返回
	// When fn is nil, return directly.
	if fn == nil {
		return nil
	}

	// t 用于定时重试
	// t is used for timing retry.
	t := time.NewTimer(r.config.delay)
	defer t.Stop()

	// 执行结果
	// Execution result.
	result := NewResult()

	// 重试逻辑
	// Retry logic.
	for {
		select {
		// ctx 被取消时直接返回
		// When ctx is canceled, return directly.
		case <-r.config.ctx.Done():
			// ctx 被取消时，返回最后一个执行过程中的错误
			// When ctx is canceled, return the last error during the execution.
			result.tryError = r.config.ctx.Err()
			return result
		// 定时器到期时执行
		// Execute when the timer expires.
		case <-t.C:
			// 执行 fn
			// Execute fn.
			d, err := fn()

			// 更新重试次数
			// Update the number of retries.
			result.count++

			// 如果执行成功，则直接返回
			// If the execution is successful, return directly.
			if err == nil {
				result.data = d
				result.tryError = err
				return result
			}

			// 记录执行过程中的错误
			// Record the error during the execution.
			if r.config.detail {
				result.execErrors = append(result.execErrors, err)
			}

			// 使用 retryIf 函数，判断是否需要重试
			// Use the retryIf function to determine whether to retry.
			if !r.config.retryIf(err) {
				result.tryError = ErrorRetryIf
				return result
			}

			// 计算下一次重试的延迟时间
			// Calculate the delay time for the next retry.
			delay := int64(rand.Float64()*float64(r.config.jitter) + float64(result.count)*r.config.factor)
			// 如果延迟时间小于等于 0，则使用默认延迟时间
			// If the delay time is less than or equal to 0, use the default delay time.
			if delay <= 0 {
				delay = defaultDelayNum
			}
			// 计算需要回退的时间
			// backoff = backoffFunc(factor * count + jitter * rand.Float64()) * 100 * Millisecond + delay
			// Calculate the time to be rolled back.
			backoff := r.config.backoff(int64(delay)) + r.config.delay

			// 执行重试回调函数
			// Execute the retry callback function.
			r.config.cb.OnRetry(int64(result.count), backoff, err)

			// 根据错误类型，判断是否需要重试。如果指定的错误次数超过限制，则直接返回
			// Determine whether to retry based on the error type. If the specified number of errors exceeds the limit, return directly.
			if errAttempts, ok := r.config.attemptsByError[err]; ok {
				if errAttempts <= 0 {
					result.tryError = ErrorRetryAttemptsByErrorExceeded
					return result
				}
				errAttempts--
				r.config.attemptsByError[err] = errAttempts
			}

			// 如果总重试次数超过限制，则直接返回
			// If the total number of retries exceeds the limit, return directly.
			if result.count >= r.config.attempts {
				result.tryError = ErrorRetryAttemptsExceeded
				return result
			}

			// 重置定时器，等待下一次重试
			// Reset the timer and wait for the next retry.
			t.Reset(backoff)
		}
	}
}

// Do 方法用于执行重试
// The Do method is used to execute the retry.
func Do(fn RetryableFunc, conf *Config) *Result {
	return New(conf).TryOnConflict(fn)
}

// DoWithDefault 方法用于执行重试，使用默认配置
// The DoWithDefault method is used to execute the retry with the default configuration.
func DoWithDefault(fn RetryableFunc) *Result {
	return New(nil).TryOnConflict(fn)
}
