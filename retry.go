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

// New 函数用于创建一个新的 Retry 实例。它接受一个 Config 结构体作为参数，该结构体包含了重试的配置信息。
// The New function is used to create a new Retry instance. It accepts a Config structure as a parameter, which contains the configuration information for retrying.
func New(conf *Config) *Retry {
	conf = isConfigValid(conf)
	return &Retry{config: conf}
}

// TryOnConflict 方法尝试执行 fn 函数，如果遇到冲突则进行重试
// The TryOnConflict method attempts to execute the fn function, and retries if a conflict is encountered
func (r *Retry) TryOnConflict(fn RetryableFunc) *Result {
	// 如果 fn 函数为空，则返回 nil。这是因为没有函数可以执行，所以没有必要进行重试。
	// If the fn function is null, return nil. This is because there is no function to execute, so there is no need to retry.
	if fn == nil {
		return nil
	}

	// 创建一个新的定时器，定时器的延迟时间是 Config 中配置的延迟时间。定时器用于控制重试的间隔。
	// Create a new timer. The delay time of the timer is the delay time configured in Config. The timer is used to control the interval between retries.
	tr := time.NewTimer(r.config.delay)

	// 使用 defer 关键字确保定时器在函数结束时停止，避免资源泄露。
	// Use the defer keyword to ensure that the timer stops when the function ends, to avoid resource leaks.
	defer tr.Stop()

	// 创建一个新的 Result 实例来存储执行结果。Result 结构体包含了执行的结果和错误信息。
	// Create a new Result instance to store the execution result. The Result structure contains the execution result and error information.
	result := NewResult()

	// 循环尝试执行 fn 函数，直到满足退出条件
	// Loop to try to execute the fn function until the exit condition is met
	for {
		select {
		// 如果上下文已完成（例如，超时或手动取消），则将上下文的错误设置为结果的错误，并返回结果
		// If the context is done (for example, timeout or manually cancelled), set the error of the context as the error of the result and return the result
		case <-r.config.ctx.Done():
			result.tryError = r.config.ctx.Err()
			return result

		// 如果定时器到时，则尝试执行 fn 函数。定时器的时间间隔由 Config 中的退避函数和抖动决定。
		// If the timer is up, try to execute the fn function. The time interval of the timer is determined by the backoff function and jitter in Config.
		case <-tr.C:
			// 调用 fn 函数，获取返回的数据和错误
			// Call the fn function to get the returned data and error
			data, err := fn()

			// 增加执行次数
			// Increase the execution count
			result.count++

			// 如果没有错误，则返回结果
			// If there is no error, return the result
			if err == nil {
				// 将数据和错误（此时为 nil）设置到结果中
				// Set the data and error (which is nil at this time) to the result
				result.data = data
				result.tryError = err

				// 返回结果
				// Return the result
				return result
			}

			// 如果需要详细信息，则添加执行错误
			// If details are needed, add execution errors
			if r.config.detail {
				// 将错误添加到结果的执行错误列表中
				// Add the error to the execution error list of the result
				result.execErrors = append(result.execErrors, err)
			}

			// 如果不需要重试，则返回结果
			// If no retry is needed, return the result
			if !r.config.retryIfFunc(err) {
				// 将错误设置到结果中
				// Set the error to the result
				result.tryError = ErrorRetryIf

				// 返回结果
				// Return the result
				return result
			}
			// 计算下一次重试的延迟时间，这里使用了一个随机的抖动和重试次数的乘积作为因子
			// Calculate the delay time for the next retry, here a random jitter and the product of the number of retries are used as factors
			delay := int64(rand.Float64()*float64(r.config.jitter) + float64(result.count)*r.config.factor)

			// 如果计算出的延迟时间小于等于 0，则设置为默认的延迟时间
			// If the calculated delay time is less than or equal to 0, set it to the default delay time
			if delay <= 0 {
				delay = defaultDelayNum
			}

			// 计算退避时间，这里使用了配置中的退避函数和延迟时间
			// Calculate the backoff time, here the backoff function and delay time in the configuration are used
			backoff := r.config.backoffFunc(int64(delay)) + r.config.delay

			// 调用配置中的回调函数，传入重试次数、退避时间和错误
			// Call the callback function in the configuration, passing in the number of retries, backoff time, and error
			r.config.callback.OnRetry(int64(result.count), backoff, err)

			// 首先，我们检查特定错误的重试次数是否已经超过限制
			// First, we check if the retry count for a specific error has exceeded the limit
			// 如果错误次数超过限制，则返回结果
			// If the number of errors exceeds the limit, return the result
			if errAttempts, ok := r.config.attemptsByError[err]; ok {
				// 如果特定错误的重试次数已经用完，则返回一个错误，表示按错误类型的重试次数已经超过
				// If the retry count for a specific error has been used up, return an error indicating that the retry count by error type has been exceeded
				if errAttempts <= 0 {
					// 将错误设置到结果中，这个错误表示特定错误的重试次数已经超过了限制
					// Set the error to the result, this error indicates that the retry count for a specific error has exceeded the limit
					result.tryError = ErrorRetryAttemptsByErrorExceeded

					// 返回结果，这个结果包含了执行的次数、最后一次的错误和尝试的错误
					// Return the result, this result includes the number of executions, the last error, and the attempted error
					return result
				}

				// 如果还有剩余的重试次数，则减少一次重试次数，并更新到配置中
				// If there are remaining retry counts, decrease the retry count by one and update it in the configuration
				errAttempts--
				r.config.attemptsByError[err] = errAttempts
			}

			// 然后，我们检查总的执行次数是否已经超过限制
			// Then, we check if the total number of executions has exceeded the limit
			// 如果执行次数超过限制，则返回结果
			// If the number of executions exceeds the limit, return the result
			if result.count >= r.config.attempts {
				// 将错误设置到结果中，这个错误表示总的执行次数已经超过了限制
				// Set the error to the result, this error indicates that the total number of executions has exceeded the limit
				result.tryError = ErrorRetryAttemptsExceeded

				// 返回结果，这个结果包含了执行的次数、最后一次的错误和尝试的错误
				// Return the result, this result includes the number of executions, the last error, and the attempted error
				return result
			}

			// 重置定时器
			// Reset the timer
			tr.Reset(backoff)
		}
	}
}

// TryOnConflict 方法尝试执行 RetryableFunc 函数，如果发生冲突，则进行重试
// The TryOnConflict method tries to execute the RetryableFunc function, and retries if a conflict occurs
func (r *Retry) TryOnConflictVal(fn RetryableFunc) RetryResult {
	return r.TryOnConflict(fn)
}

// Do 函数尝试执行 fn 函数，如果遇到冲突则根据 conf 配置进行重试
// The Do function attempts to execute the fn function, and retries according to the conf configuration if a conflict is encountered
func Do(fn RetryableFunc, conf *Config) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数
	// Create a new Retry instance and try to execute the fn function
	return New(conf).TryOnConflict(fn)
}

// DoWithDefault 函数尝试执行 fn 函数，如果遇到冲突则使用默认配置进行重试
// The DoWithDefault function attempts to execute the fn function, and retries with the default configuration if a conflict is encountered
func DoWithDefault(fn RetryableFunc) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数
	// Create a new Retry instance and try to execute the fn function
	return New(nil).TryOnConflict(fn)
}
