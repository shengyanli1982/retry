package retry

import (
	"maps"
	"math/rand/v2"
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

// ExecErrors 方法返回所有执行错误的列表（仅 Config.detail 为 true 时记录，否则为空切片）
// The ExecErrors method returns a list of all execution errors (recorded only when Config.detail
// is true; otherwise an empty slice)
func (r *Result) ExecErrors() []error {
	return r.execErrors
}

// IsSuccess 方法返回执行是否成功
// The IsSuccess method returns whether the execution was successful
func (r *Result) IsSuccess() bool {
	return r.tryError == nil
}

// LastExecError 方法返回最后一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
// The LastExecError method returns the error of the last execution; it returns ErrorExecErrNotFound
// when the error list is empty (including when detail=false recorded nothing)
func (r *Result) LastExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[len(r.execErrors)-1]
	}
	return ErrorExecErrNotFound
}

// FirstExecError 方法返回第一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
// The FirstExecError method returns the error of the first execution; it returns ErrorExecErrNotFound
// when the error list is empty (including when detail=false recorded nothing)
func (r *Result) FirstExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[0]
	}
	return ErrorExecErrNotFound
}

// ExecErrorByIndex 方法返回指定索引处的执行错误；索引越界时返回 ErrorExecErrByIndexOutOfBound
// The ExecErrorByIndex method returns the execution error at the specified index; it returns
// ErrorExecErrByIndexOutOfBound when the index is out of range
func (r *Result) ExecErrorByIndex(idx int) error {
	if idx >= 0 && idx < len(r.execErrors) {
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
// New 对传入的 Config 做浅拷贝，归一化只作用于副本，永不写调用方对象（避免副作用外泄与并发写竞争）。
// The New function is used to create a new Retry instance. It accepts a Config structure as a parameter, which contains the configuration information for retrying.
// New shallow-copies the passed Config; normalization applies only to the copy and never writes the caller's object (avoiding side-effect leakage and concurrent-write races).
func New(conf *Config) *Retry {
	if conf != nil {
		c := *conf
		conf = &c
	}
	// conf==nil 时 isConfigValid 返回全新的默认 Config，语义保持不变
	// When conf==nil, isConfigValid returns a brand-new default Config; the semantics are unchanged
	return &Retry{config: isConfigValid(conf)}
}

// TryOnConflict 方法尝试执行 fn 函数，如果遇到冲突则进行重试。
// "OnConflict" 的含义：fn 返回的任何非 nil 错误都视为一次冲突，触发按配置的重试。
// fn 为 nil 时返回 nil；因错误中止时 TryError 以双 %w 同时包装哨兵错误与根因（见 error.go）。
// The TryOnConflict method attempts to execute the fn function, and retries if a conflict is encountered.
// Meaning of "OnConflict": any non-nil error returned by fn is treated as a conflict that triggers a
// configured retry. It returns nil when fn is nil; on an error-triggered abort, TryError wraps both the sentinel error
// and the root cause with double %w (see error.go).
func (r *Retry) TryOnConflict(fn RetryableFunc) *Result {
	// 如果 fn 函数为空，则返回 nil。这是因为没有函数可以执行，所以没有必要进行重试。
	// If the fn function is null, return nil. This is because there is no function to execute, so there is no need to retry.
	if fn == nil {
		return nil
	}

	// 创建 attemptsByError 的本地副本，避免并发访问共享 map 导致 data race。
	// 空 map 与 nil map 在 ok 守卫（if errAttempts, ok := localAttemptsByError[err]; ok）下行为完全一致
	// （nil map 读返回零值 + ok=false），因此空 map 时跳过 Clone 以消除一次堆分配。
	// Create a local copy of attemptsByError to avoid data race when accessing the shared map concurrently.
	// An empty map and a nil map behave identically under the ok guard
	// (nil map read returns zero value + ok=false), so skip Clone for empty maps to eliminate a heap allocation.
	var localAttemptsByError map[error]uint64
	if len(r.config.attemptsByError) > 0 {
		localAttemptsByError = maps.Clone(r.config.attemptsByError)
	}

	// 创建一个新的 Result 实例来存储执行结果。Result 结构体包含了执行的结果和错误信息。
	// Create a new Result instance to store the execution result. The Result structure contains the execution result and error information.
	result := NewResult()

	// ── 首次执行：直接执行，不创建 timer，消除成功路径上的 timer 分配开销 ──
	// ── First execution: execute directly without creating a timer, eliminating timer allocation overhead on the success path ──

	// 非阻塞检查 context 是否已取消
	// Non-blocking check if context is already cancelled
	select {
	case <-r.config.ctx.Done():
		result.tryError = r.config.ctx.Err()
		return result
	default:
	}

	// 调用 fn 函数，获取返回的数据和错误
	// Call the fn function to get the returned data and error
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
		result.tryError = &wrappedError{sentinel: ErrorRetryIf, cause: err}
		return result
	}

	// 检查特定错误的重试次数是否已经超过限制
	// Check if the retry count for a specific error has exceeded the limit
	if errAttempts, ok := localAttemptsByError[err]; ok {
		if errAttempts <= 0 {
			result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsByErrorExceeded, cause: err}
			return result
		}
		errAttempts--
		localAttemptsByError[err] = errAttempts
	}

	// 检查总的执行次数是否已经超过限制
	// Check if the total number of executions has exceeded the limit
		if result.count >= r.config.attempts {
			result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsExceeded, cause: err}
		return result
	}

	// 计算下一次重试的延迟并创建 timer（仅首次失败后才创建）
	// Calculate the delay for the next retry and create the timer (only created after first failure)
	// rand/v2 顶层 Float64 goroutine 安全且 per-M 无锁，值域 [0,1) 与原实现一致
	// rand/v2 top-level Float64 is goroutine-safe and per-M lock-free; its [0,1) range matches the previous implementation
	jitterVal := rand.Float64()
	delay := time.Duration(jitterVal*r.config.jitter+float64(result.count)*r.config.factor) * r.config.delay
	backoff := r.config.backoffFunc(int64(result.count)) + delay

	// 通过全部中止检查后、实际发起下一次重试前，才调用回调函数
	// Only after passing all abort checks and before actually starting the next retry, call the callback
	r.config.callback.OnRetry(int64(result.count), backoff, err)

	// 创建定时器，用于控制后续重试的间隔
	// Create a timer to control the interval between subsequent retries
	tr := time.NewTimer(backoff)
	defer tr.Stop()

	// ── 后续重试循环 ──
	// ── Subsequent retry loop ──
	for {
		// 非阻塞检查 context 是否已取消
		// Non-blocking check if context is already cancelled
		select {
		case <-r.config.ctx.Done():
			result.tryError = r.config.ctx.Err()
			return result
		default:
		}

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
				result.tryError = &wrappedError{sentinel: ErrorRetryIf, cause: err}
				return result
			}

			// 计算下一次重试的线性延迟部分：随机抖动 + 重试次数 * 因子，再乘以基础延迟时间
			// Calculate the linear delay component for the next retry: random jitter + retry count * factor, then multiply by base delay
			jitterVal := rand.Float64()
			delay := time.Duration(jitterVal*r.config.jitter+float64(result.count)*r.config.factor) * r.config.delay

			// 计算退避时间：退避函数接收当前重试次数作为参数，加上线性延迟部分
			// Calculate the backoff time: backoff function receives the current retry count as parameter, plus the linear delay component
			backoff := r.config.backoffFunc(int64(result.count)) + delay

			// 检查特定错误的重试次数是否已经超过限制
			// Check if the retry count for a specific error has exceeded the limit
			if errAttempts, ok := localAttemptsByError[err]; ok {
				if errAttempts <= 0 {
					result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsByErrorExceeded, cause: err}
					return result
				}
				errAttempts--
				localAttemptsByError[err] = errAttempts
			}

			// 检查总的执行次数是否已经超过限制
			// Check if the total number of executions has exceeded the limit
			if result.count >= r.config.attempts {
				result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsExceeded, cause: err}
				return result
			}

			// 通过全部中止检查后、实际发起下一次重试前，才调用回调函数
			// Only after passing all abort checks and before actually starting the next retry, call the callback
			r.config.callback.OnRetry(int64(result.count), backoff, err)

			// 重置定时器
			// Reset the timer
			tr.Reset(backoff)
		}
	}
}

// TryOnConflictVal 方法与 TryOnConflict 相同，但返回 RetryResult 接口。
// The TryOnConflictVal method is identical to TryOnConflict but returns the RetryResult interface.
func (r *Retry) TryOnConflictVal(fn RetryableFunc) RetryResult {
	// fn==nil 时内部返回 nil *Result，不得装箱进接口（typed-nil 会使调用方 == nil 判空失效）
	// When fn==nil the inner call returns a nil *Result, which must not be boxed into the interface (a typed-nil breaks the caller's == nil check)
	if res := r.TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}

// Do 函数使用指定配置执行 fn 函数，如果遇到冲突则按配置进行重试；conf 为 nil 时使用默认配置。
// fn 为 nil 时返回真 nil 接口（而非装箱的 typed-nil）。
// The Do function executes the fn function with the given config, and retries according to the conf
// configuration if a conflict is encountered; a nil conf means the default configuration.
// It returns a true nil interface (not a boxed typed-nil) when fn is nil.
func Do(fn RetryableFunc, conf *Config) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数；nil *Result 不装箱进接口
	// Create a new Retry instance and try to execute the fn function; a nil *Result is not boxed into the interface
	if res := New(conf).TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}

// DoWithDefault 函数使用默认配置执行 fn 函数，如果遇到冲突则进行重试；"Default" 指 NewConfig 的全套默认值（等价于 Do(fn, nil)）。
// fn 为 nil 时返回真 nil 接口（而非装箱的 typed-nil）。
// The DoWithDefault function executes the fn function with the default configuration, and retries if a
// conflict is encountered; "Default" refers to the full set of NewConfig defaults (equivalent to Do(fn, nil)).
// It returns a true nil interface (not a boxed typed-nil) when fn is nil.
func DoWithDefault(fn RetryableFunc) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数；nil *Result 不装箱进接口
	// Create a new Retry instance and try to execute the fn function; a nil *Result is not boxed into the interface
	if res := New(nil).TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}
