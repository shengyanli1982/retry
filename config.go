package retry

import (
	"context"
	"math"
	"time"
)

// 定义默认的重试次数、延迟时间、抖动和因子
// Define the default number of retries, delay time, jitter, and factor
const (
	defaultAttempts = 3                                        // 默认的重试次数为3次
	defaultDelayNum = 5                                        // 默认的延迟基数为5（对应5个100毫秒基础时间单位）
	defaultDelay    = defaultDelayNum * time.Millisecond * 100 // 默认的延迟时间：5 * 1ms * 100 = 500毫秒
	defaultJitter   = 3.0                                      // 默认的抖动为3.0
	defaultFactor   = 1.0                                      // 默认的因子为1.0
)

// 定义默认的重试条件函数和退避函数
// Define the default retry condition function and backoff function
var (
	// defaultRetryIfFunc 是默认的重试条件函数，对所有错误都进行重试
	// defaultRetryIfFunc is the default retry condition function, which retries for all errors
	defaultRetryIfFunc = func(error) bool { return true }

	// defaultBackoffFunc 是默认的退避函数，使用指数退避和随机退避的组合。
	// 包级一次性构造：闭包无状态（随机数仅在 randMu 保护下读取全局 randGen），并发安全，避免每次调用重建组合闭包。
	// defaultBackoffFunc is the default backoff function, which combines exponential backoff and random backoff.
	// Constructed once at package level: the closure is stateless (random reads only touch the global randGen under randMu) and concurrency-safe, avoiding rebuilding the combined closure on every call.
	defaultBackoffFunc = CombineBackoffs(ExponentialBackoff, RandomBackoff)
)

// 定义一个空的回调结构体
// Define an empty callback structure
type emptyCallback struct{}

// OnRetry 方法在每次重试时调用，但不执行任何操作
// The OnRetry method is called on each retry, but does not perform any operations
func (cb *emptyCallback) OnRetry(count int64, delay time.Duration, err error) {}

// NewEmptyCallback 函数返回一个新的空回调实例
// The NewEmptyCallback function returns a new empty callback instance
func NewEmptyCallback() Callback {
	return &emptyCallback{}
}

// RetryIfFunc 类型定义了一个接受错误并返回布尔值的函数类型
// The RetryIfFunc type defines a function type that accepts an error and returns a boolean value
type RetryIfFunc = func(error) bool

// Config 结构体定义了重试的配置
// The Config structure defines the configuration for retries
type Config struct {
	ctx             context.Context  // 上下文，用于控制重试的生命周期
	callback        Callback         // 回调函数，用于在每次重试时执行
	attempts        uint64           // 重试次数
	attemptsByError map[error]uint64 // 按错误类型的重试次数
	factor          float64          // 退避因子，用于控制退避时间的增长速度
	jitter          float64          // 抖动，用于在退避时间上添加随机性
	delay           time.Duration    // 延迟时间，用于控制每次重试之间的间隔
	retryIfFunc     RetryIfFunc      // 重试条件函数，用于判断是否应该重试
	backoffFunc     BackoffFunc      // 退避函数，用于计算每次重试的延迟时间
	detail          bool             // 是否显示详细的错误信息
}

// NewConfig 函数返回一个新的 Config 实例，使用默认的配置
// The NewConfig function returns a new Config instance with the default configuration
func NewConfig() *Config {
	return &Config{
		ctx:             context.Background(),
		callback:        NewEmptyCallback(),
		attempts:        defaultAttempts,
		attemptsByError: make(map[error]uint64),
		factor:          defaultFactor,
		delay:           defaultDelay,
		jitter:          defaultJitter,
		retryIfFunc:     defaultRetryIfFunc,
		backoffFunc:     defaultBackoffFunc,
		detail:          false,
	}
}

// WithContext 方法设置 Config 的上下文并返回 Config 实例
// The WithContext method sets the context of the Config and returns the Config instance
func (c *Config) WithContext(ctx context.Context) *Config {
	c.ctx = ctx
	return c
}

// WithCallback 方法设置 Config 的回调函数并返回 Config 实例
// The WithCallback method sets the callback function of the Config and returns the Config instance
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

// WithAttempts 方法设置 Config 的重试次数并返回 Config 实例。
// 取值在 New 时归一化：0 → 默认 3；>= 65535 → 钳制为 65534（见 isConfigValid）。
// The WithAttempts method sets the number of retries of the Config and returns the Config instance.
// The value is normalized by New: 0 falls back to the default 3; values >= 65535 are clamped to 65534 (see isConfigValid).
func (c *Config) WithAttempts(attempts uint64) *Config {
	c.attempts = attempts
	return c
}

// WithAttemptsByError 方法设置 Config 的错误重试次数并返回 Config 实例。
// 入口拷贝用户 map，库持有私有副本：用户后续对原 map 的写入不影响重试预算，
// 也避免重试执行期间与用户并发写相遇导致 fatal error: concurrent map read and map write。
// The WithAttemptsByError method sets the number of error retries of the Config and returns the Config instance.
// It copies the user map at the entry so the library holds a private copy: later writes to the original map do not affect the retry budget,
// and a concurrent map read/write fatal error between retry execution and user writes is avoided.
func (c *Config) WithAttemptsByError(attemptsByError map[error]uint64) *Config {
	c.attemptsByError = make(map[error]uint64, len(attemptsByError))
	for k, v := range attemptsByError {
		c.attemptsByError[k] = v
	}
	return c
}

// WithFactor 方法设置 Config 的因子并返回 Config 实例
// The WithFactor method sets the factor of the Config and returns the Config instance
func (c *Config) WithFactor(factor float64) *Config {
	c.factor = factor
	return c
}

// WithInitDelay 方法设置 Config 的基础延迟（每次重试线性延迟项的乘数基数，见 retry.go 的退避计算）并返回 Config 实例
// The WithInitDelay method sets the base delay of the Config (the multiplier base of the linear delay
// component for each retry, see the backoff calculation in retry.go) and returns the Config instance
func (c *Config) WithInitDelay(delay time.Duration) *Config {
	c.delay = delay
	return c
}

// WithJitter 方法设置 Config 的抖动并返回 Config 实例
// The WithJitter method sets the jitter of the Config and returns the Config instance
func (c *Config) WithJitter(jitter float64) *Config {
	c.jitter = jitter
	return c
}

// WithRetryIfFunc 方法设置 Config 的重试条件函数并返回 Config 实例
// The WithRetryIfFunc method sets the retry condition function of the Config and returns the Config instance
func (c *Config) WithRetryIfFunc(retryIf RetryIfFunc) *Config {
	c.retryIfFunc = retryIf
	return c
}

// WithBackOffFunc 方法设置 Config 的退避函数并返回 Config 实例
// The WithBackOffFunc method sets the backoff function of the Config and returns the Config instance
func (c *Config) WithBackOffFunc(backoff BackoffFunc) *Config {
	c.backoffFunc = backoff
	return c
}

// WithDetail 方法设置 Config 的详细错误信息显示选项并返回 Config 实例
// The WithDetail method sets the detailed error information display option of the Config and returns the Config instance
func (c *Config) WithDetail(detail bool) *Config {
	c.detail = detail
	return c
}

// isConfigValid 函数检查 Config 是否有效，如果无效则使用默认值
// The isConfigValid function checks whether the Config is valid, and uses the default value if it is invalid
func isConfigValid(conf *Config) *Config {
	// 如果 conf 为 nil，则创建一个新的 Config 实例
	// If conf is nil, create a new Config instance
	if conf == nil {
		conf = NewConfig()
	} else {
		// 如果 conf.ctx 为 nil，则设置为默认的上下文
		// If conf.ctx is nil, set it to the default context
		if conf.ctx == nil {
			conf.ctx = context.Background()
		}

		// 如果 conf.callback 为 nil，则设置为默认的回调函数
		// If conf.callback is nil, set it to the default callback function
		if conf.callback == nil {
			conf.callback = NewEmptyCallback()
		}

		// 如果 conf.attempts 为 0，则重置为默认的重试次数；如果达到或超过 math.MaxUint16（65535），则钳制为 math.MaxUint16-1（65534）。
		// 上限钳制而非重置的原因：65534 是可表示的最大合法重试预算，把巨大值静默重置为默认 3 会造成"要求更多却得到更少"的反直觉语义。
		// If conf.attempts is 0, reset it to the default number of retries; if it reaches or exceeds math.MaxUint16 (65535), clamp it to math.MaxUint16-1 (65534).
		// Clamping instead of resetting: 65534 is the largest representable valid retry budget, while silently resetting huge values to the default 3 creates the counter-intuitive "ask for more, get less" semantics.
		if conf.attempts <= 0 {
			conf.attempts = defaultAttempts
		} else if conf.attempts >= math.MaxUint16 {
			conf.attempts = math.MaxUint16 - 1
		}

		// 如果 conf.attemptsByError 为 nil，则初始化为一个空的映射
		// If conf.attemptsByError is nil, initialize it to an empty map
		if conf.attemptsByError == nil {
			conf.attemptsByError = make(map[error]uint64)
		}

		// 如果 conf.factor 小于 0，则设置为默认的退避因子
		// If conf.factor is less than 0, set it to the default backoff factor
		if conf.factor < 0 {
			conf.factor = defaultFactor
		}

		// 如果 conf.delay 小于等于 0，则设置为默认的延迟时间
		// If conf.delay is less than or equal to 0, set it to the default delay time
		if conf.delay <= 0 {
			conf.delay = defaultDelay
		}

		// 如果 conf.jitter 小于 0，则设置为默认的抖动
		// If conf.jitter is less than 0, set it to the default jitter
		if conf.jitter < 0 {
			conf.jitter = defaultJitter
		}

		// 如果 conf.retryIfFunc 为 nil，则设置为默认的重试条件函数
		// If conf.retryIfFunc is nil, set it to the default retry condition function
		if conf.retryIfFunc == nil {
			conf.retryIfFunc = defaultRetryIfFunc
		}

		// 如果 conf.backoffFunc 为 nil，则设置为默认的退避函数
		// If conf.backoffFunc is nil, set it to the default backoff function
		if conf.backoffFunc == nil {
			conf.backoffFunc = defaultBackoffFunc
		}
	}

	// 返回检查并修正后的 Config 实例
	// Return the checked and corrected Config instance
	return conf
}

// DefaultConfig 函数返回一个新的默认配置的 Config 实例
// The DefaultConfig function returns a new Config instance with the default configuration
func DefaultConfig() *Config {
	return NewConfig()
}

// FixConfig 函数返回一个新的固定退避时间的 Config 实例：每次重试固定间隔 500ms
// （backoffFunc 恒返回 defaultDelay，factor=0、jitter=0 使线性延迟项为 0）
// The FixConfig function returns a new Config instance with a fixed backoff time: every retry waits a
// constant 500ms (the backoffFunc always returns defaultDelay, and factor=0, jitter=0 zero out the linear delay component)
func FixConfig() *Config {
	return NewConfig().WithBackOffFunc(func(_ int64) time.Duration {
		return defaultDelay
	}).WithFactor(0).WithJitter(0)
}
