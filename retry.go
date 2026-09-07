package retry

import (
	"maps"
	"math/rand/v2"
	"time"
)

// Result 结构体用于存储执行结果
type Result struct {
	count      uint64  // 执行次数
	data       any     // 执行结果数据
	tryError   error   // 尝试执行时的错误
	execErrors []error // 执行错误列表
}

// NewResult 函数用于创建一个新的 Result 实例
func NewResult() *Result {
	return &Result{execErrors: make([]error, 0)}
}

// Data 方法返回执行结果的数据
func (r *Result) Data() any {
	return r.data
}

// TryError 方法返回尝试执行时的错误
func (r *Result) TryError() error {
	return r.tryError
}

// ExecErrors 方法返回所有执行错误的列表（仅 Config.detail 为 true 时记录，否则为空切片）
func (r *Result) ExecErrors() []error {
	return r.execErrors
}

// IsSuccess 方法返回执行是否成功
func (r *Result) IsSuccess() bool {
	return r.tryError == nil
}

// LastExecError 方法返回最后一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
func (r *Result) LastExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[len(r.execErrors)-1]
	}
	return ErrorExecErrNotFound
}

// FirstExecError 方法返回第一次执行的错误；错误列表为空（含 detail=false 未记录）时返回 ErrorExecErrNotFound
func (r *Result) FirstExecError() error {
	if len(r.execErrors) > 0 {
		return r.execErrors[0]
	}
	return ErrorExecErrNotFound
}

// ExecErrorByIndex 方法返回指定索引处的执行错误；索引越界时返回 ErrorExecErrByIndexOutOfBound
func (r *Result) ExecErrorByIndex(idx int) error {
	if idx >= 0 && idx < len(r.execErrors) {
		return r.execErrors[idx]
	}
	return ErrorExecErrByIndexOutOfBound
}

// Count 方法返回执行的次数
func (r *Result) Count() int64 {
	return int64(r.count)
}

// RetryableFunc 类型定义了一个可重试的函数
type RetryableFunc = func() (any, error)

// Retry 结构体用于定义重试的配置
type Retry struct {
	config *Config // 重试的配置
}

// New 函数用于创建一个新的 Retry 实例。它接受一个 Config 结构体作为参数，该结构体包含了重试的配置信息。
// New 对传入的 Config 做浅拷贝，归一化只作用于副本，永不写调用方对象（避免副作用外泄与并发写竞争）。
func New(conf *Config) *Retry {
	if conf != nil {
		c := *conf
		conf = &c
	}
	// conf==nil 时 isConfigValid 返回全新的默认 Config，语义保持不变
	return &Retry{config: isConfigValid(conf)}
}

// TryOnConflict 方法尝试执行 fn 函数，如果遇到冲突则进行重试。
// "OnConflict" 的含义：fn 返回的任何非 nil 错误都视为一次冲突，触发按配置的重试。
// fn 为 nil 时返回 nil；因错误中止时 TryError 以双 %w 同时包装哨兵错误与根因（见 error.go）。
func (r *Retry) TryOnConflict(fn RetryableFunc) *Result {
	// 如果 fn 函数为空，则返回 nil。这是因为没有函数可以执行，所以没有必要进行重试。
	if fn == nil {
		return nil
	}

	// 创建 attemptsByError 的本地副本，避免并发访问共享 map 导致 data race。
	// 空 map 与 nil map 在 ok 守卫（if errAttempts, ok := localAttemptsByError[err]; ok）下行为完全一致
	// （nil map 读返回零值 + ok=false），因此空 map 时跳过 Clone 以消除一次堆分配。
	var localAttemptsByError map[error]uint64
	if len(r.config.attemptsByError) > 0 {
		localAttemptsByError = maps.Clone(r.config.attemptsByError)
	}

	// 创建一个新的 Result 实例来存储执行结果。Result 结构体包含了执行的结果和错误信息。
	result := NewResult()

	// ── 首次执行：直接执行，不创建 timer，消除成功路径上的 timer 分配开销 ──

	// 非阻塞检查 context 是否已取消
	select {
	case <-r.config.ctx.Done():
		result.tryError = r.config.ctx.Err()
		return result
	default:
	}

	// 调用 fn 函数，获取返回的数据和错误
	data, err := fn()

	// 增加执行次数
	result.count++

	// 如果没有错误，则返回结果
	if err == nil {
		result.data = data
		result.tryError = err
		return result
	}

	// 如果需要详细信息，则添加执行错误
	if r.config.detail {
		result.execErrors = append(result.execErrors, err)
	}

	// 如果不需要重试，则返回结果
	if !r.config.retryIfFunc(err) {
		result.tryError = &wrappedError{sentinel: ErrorRetryIf, cause: err}
		return result
	}

	// 检查特定错误的重试次数是否已经超过限制
	if errAttempts, ok := localAttemptsByError[err]; ok {
		if errAttempts <= 0 {
			result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsByErrorExceeded, cause: err}
			return result
		}
		errAttempts--
		localAttemptsByError[err] = errAttempts
	}

	// 检查总的执行次数是否已经超过限制
		if result.count >= r.config.attempts {
			result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsExceeded, cause: err}
		return result
	}

	// 计算下一次重试的延迟并创建 timer（仅首次失败后才创建）
	// rand/v2 顶层 Float64 goroutine 安全且 per-M 无锁，值域 [0,1) 与原实现一致
	jitterVal := rand.Float64()
	delay := time.Duration(jitterVal*r.config.jitter+float64(result.count)*r.config.factor) * r.config.delay
	backoff := r.config.backoffFunc(int64(result.count)) + delay

	// 通过全部中止检查后、实际发起下一次重试前，才调用回调函数
	r.config.callback.OnRetry(int64(result.count), backoff, err)

	// 创建定时器，用于控制后续重试的间隔
	tr := time.NewTimer(backoff)
	defer tr.Stop()

	// ── 后续重试循环 ──
	for {
		// 非阻塞检查 context 是否已取消
		select {
		case <-r.config.ctx.Done():
			result.tryError = r.config.ctx.Err()
			return result
		default:
		}

		select {
		// 如果上下文已完成（例如，超时或手动取消），则将上下文的错误设置为结果的错误，并返回结果
		case <-r.config.ctx.Done():
			result.tryError = r.config.ctx.Err()
			return result

		// 如果定时器到时，则尝试执行 fn 函数。定时器的时间间隔由 Config 中的退避函数和抖动决定。
		case <-tr.C:
			// 调用 fn 函数，获取返回的数据和错误
			data, err := fn()

			// 增加执行次数
			result.count++

			// 如果没有错误，则返回结果
			if err == nil {
				result.data = data
				result.tryError = err
				return result
			}

			// 如果需要详细信息，则添加执行错误
			if r.config.detail {
				result.execErrors = append(result.execErrors, err)
			}

			// 如果不需要重试，则返回结果
			if !r.config.retryIfFunc(err) {
				result.tryError = &wrappedError{sentinel: ErrorRetryIf, cause: err}
				return result
			}

			// 计算下一次重试的线性延迟部分：随机抖动 + 重试次数 * 因子，再乘以基础延迟时间
			jitterVal := rand.Float64()
			delay := time.Duration(jitterVal*r.config.jitter+float64(result.count)*r.config.factor) * r.config.delay

			// 计算退避时间：退避函数接收当前重试次数作为参数，加上线性延迟部分
			backoff := r.config.backoffFunc(int64(result.count)) + delay

			// 检查特定错误的重试次数是否已经超过限制
			if errAttempts, ok := localAttemptsByError[err]; ok {
				if errAttempts <= 0 {
					result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsByErrorExceeded, cause: err}
					return result
				}
				errAttempts--
				localAttemptsByError[err] = errAttempts
			}

			// 检查总的执行次数是否已经超过限制
			if result.count >= r.config.attempts {
				result.tryError = &wrappedError{sentinel: ErrorRetryAttemptsExceeded, cause: err}
				return result
			}

			// 通过全部中止检查后、实际发起下一次重试前，才调用回调函数
			r.config.callback.OnRetry(int64(result.count), backoff, err)

			// 重置定时器
			tr.Reset(backoff)
		}
	}
}

// TryOnConflictVal 方法与 TryOnConflict 相同，但返回 RetryResult 接口。
func (r *Retry) TryOnConflictVal(fn RetryableFunc) RetryResult {
	// fn==nil 时内部返回 nil *Result，不得装箱进接口（typed-nil 会使调用方 == nil 判空失效）
	if res := r.TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}

// Do 函数使用指定配置执行 fn 函数，如果遇到冲突则按配置进行重试；conf 为 nil 时使用默认配置。
// fn 为 nil 时返回真 nil 接口（而非装箱的 typed-nil）。
func Do(fn RetryableFunc, conf *Config) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数；nil *Result 不装箱进接口
	if res := New(conf).TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}

// DoWithDefault 函数使用默认配置执行 fn 函数，如果遇到冲突则进行重试；"Default" 指 NewConfig 的全套默认值（等价于 Do(fn, nil)）。
// fn 为 nil 时返回真 nil 接口（而非装箱的 typed-nil）。
func DoWithDefault(fn RetryableFunc) RetryResult {
	// 创建一个新的 Retry 实例并尝试执行 fn 函数；nil *Result 不装箱进接口
	if res := New(nil).TryOnConflict(fn); res != nil {
		return res
	}
	return nil
}
