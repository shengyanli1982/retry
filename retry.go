package retry

import (
	"math/rand"
	"time"
)

// Result 结构体用于存储执行结果
// The Result struct is used to store the execution result
type Result struct {
	count      uint64  // 执行次数 Execution count
	data       any     // 执行结果数据 Execution result data
	tryError   error   // 尝试执行时的错误 Error when trying to execute
	execErrors []error // 执行错误列表 List of execution errors
}

// NewResult 函数用于创建一个新的 Result 实例
// The NewResult function is used to create a new Result instance
func NewResult() *Result {
	return &Result{execErrors: make([]error, 0)}
}

// Data 方法返回执行结果的数据
// The Data method returns the data of the execution result
func (r *Result) Data() any {
	return r.data
}

// TryError 方法返回尝试执行时的错误
// The TryError method returns the error when trying to execute
func (r *Result) TryError() error {
	return r.tryError
}

// ExecErrors 方法返回所有执行错误的列表
// The ExecErrors method returns a list of all execution errors
func (r *Result) ExecErrors() []error {
	return r.execErrors
}

// IsSuccess 方法返回执行是否成功
// The IsSuccess method returns whether the execution was successful
func (r *Result) IsSuccess() bool {
	return r.tryError == nil
}

// LastExecError 方法返回最后一次执行的错误
// The LastExecError method returns the error of the last execution
func (r *Result) LastExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[len(r.execErrors)-1]
	}
	return ErrorExecErrNotFound
}

// FirstExecError 方法返回第一次执行的错误
// The FirstExecError method returns the error of the first execution
func (r *Result) FirstExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[0]
	}
	return ErrorExecErrNotFound
}

// ExecErrorByIndex 方法返回指定索引处的执行错误
// The ExecErrorByIndex method returns the execution error at the specified index
func (r *Result) ExecErrorByIndex(idx int) error {
	if len(r.execErrors) >= 0 && idx < len(r.execErrors) {
		return r.execErrors[idx]
	}
	return ErrorExecErrByIndexOutOfBound
}

// Count 方法返回执行的次数
// The Count method returns the number of executions
func (r *Result) Count() int64 {
	return int64(r.count)
}

// RetryableFunc 类型定义了一个可重试的函数
// The RetryableFunc type defines a retryable function
type RetryableFunc = func() (any, error)

// Retry 结构体用于定义重试的配置
// The Retry struct is used to define the retry configuration
type Retry struct {
	config *Config // 重试的配置 Retry configuration
}

// New 函数用于创建一个新的 Retry 实例
// The New function is used to create a new Retry instance
func New(conf *Config) *Retry {
	conf = isConfigValid(conf)
	return &Retry{config: conf}
}

// TryOnConflict 方法尝试执行 fn 函数，如果遇到冲突则进行重试
// The TryOnConflict method attempts to execute the fn function, and retries if a conflict is encountered
func (r *Retry) TryOnConflict(fn RetryableFunc) *Result {
	// 如果 fn 函数为空，则返回 nil
	// If the fn function is null, return nil
	if fn == nil {
		return nil
	}

	// 创建一个新的定时器
	// Create a new timer
	tr := time.NewTimer(r.config.delay)
	defer tr.Stop()

	// 创建一个新的 Result 实例来存储执行结果
	// Create a new Result instance to store the execution result
	result := NewResult()

	// 循环尝试执行 fn 函数
	// Loop to try to execute the fn function
	for {
		select {
		// 如果上下文已完成，则返回结果
		// If the context is done, return the result
		case <-r.config.ctx.Done():
			result.tryError = r.config.ctx.Err()
			return result
		// 如果定时器到时，则尝试执行 fn 函数
		// If the timer is up, try to execute the fn function
		case <-tr.C:
			data, err := fn()

			// 增加执行次数
			// Increase the execution count
			result.count++

			// 如果没有错误，则返回结果
			// If there is no error, return the result
			if err == nil {
				result.data = data
				result.tryError = err
				return result
			}

			// 如果需要详细信息，则添加执行错误
			// If details are needed, add execution errors
			if r.config.detail {
				result.execErrors = append(result.execErrors, err)
			}

			// 如果不需要重试，则返回结果
			// If no retry is needed, return the result
			if !r.config.retryIfFunc(err) {
				result.tryError = ErrorRetryIf
				return result
			}

			// 计算下一次重试的延迟时间
			// Calculate the delay time for the next retry
			delay := int64(rand.Float64()*float64(r.config.jitter) + float64(result.count)*r.config.factor)

			// 如果延迟时间小于等于 0，则设置为默认延迟时间
			// If the delay time is less than or equal to 0, set it to the default delay time
			if delay <= 0 {
				delay = defaultDelayNum
			}

			// 计算退避时间并调用回调函数
			// Calculate the backoff time and call the callback function
			backoff := r.config.backoffFunc(int64(delay)) + r.config.delay
			r.config.callback.OnRetry(int64(result.count), backoff, err)

			// 如果错误次数超过限制，则返回结果
			// If the number of errors exceeds the limit, return the result
			if errAttempts, ok := r.config.attemptsByError[err]; ok {
				if errAttempts <= 0 {
					result.tryError = ErrorRetryAttemptsByErrorExceeded
					return result
				}
				errAttempts--
				r.config.attemptsByError[err] = errAttempts
			}

			// 如果执行次数超过限制，则返回结果
			// If the number of executions exceeds the limit, return the result
			if result.count >= r.config.attempts {
				result.tryError = ErrorRetryAttemptsExceeded
				return result
			}

			// 重置定时器
			// Reset the timer
			tr.Reset(backoff)
		}
	}
}

// TryOnConflictInterface 方法尝试执行 fn 函数，如果遇到冲突则进行重试, 返回结果接口
// The TryOnConflictInterface method attempts to execute the fn function, and retries if a conflict is encountered, returning the result interface
func (r *Retry) TryOnConflictInterface(fn RetryableFunc) ResultInterface {
	// 调用 TryOnConflict 方法执行 fn 函数并返回结果
	// Call the TryOnConflict method to execute the fn function and return the result
	return r.TryOnConflict(fn)
}

// Do 函数尝试执行 fn 函数，如果遇到冲突则根据 conf 配置进行重试
// The Do function attempts to execute the fn function, and retries according to the conf configuration if a conflict is encountered
func Do(fn RetryableFunc, conf *Config) *Result {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数
	// Create a new Retry instance and try to execute the fn function
	return New(conf).TryOnConflict(fn)
}

// DoWithDefault 函数尝试执行 fn 函数，如果遇到冲突则使用默认配置进行重试
// The DoWithDefault function attempts to execute the fn function, and retries with the default configuration if a conflict is encountered
func DoWithDefault(fn RetryableFunc) *Result {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数
	// Create a new Retry instance and try to execute the fn function
	return New(nil).TryOnConflict(fn)
}
