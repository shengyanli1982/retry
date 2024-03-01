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
	defaultDelayNum = 5                                        // 默认的延迟时间为5毫秒
	defaultDelay    = defaultDelayNum * time.Millisecond * 100 // 计算默认的延迟时间
	defaultJitter   = 3.0                                      // 默认的抖动为3.0
	defaultFactor   = 1.0                                      // 默认的因子为1.0
)

// 定义默认的重试条件函数和退避函数
// Define the default retry condition function and backoff function
var (
	defaultRetryIfFunc = func(error) bool { return true } // 默认的重试条件函数，对所有错误都进行重试
	defaultBackoffFunc = func(n int64) time.Duration {    // 默认的退避函数，使用指数退避和随机退避的组合
		return CombineBackOffs(ExponentialBackOff, RandomBackOff)(n)
	}
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

// WithAttempts 方法设置 Config 的重试次数并返回 Config 实例
// The WithAttempts method sets the number of retries of the Config and returns the Config instance
func (c *Config) WithAttempts(attempts uint64) *Config {
	c.attempts = attempts
	return c
}

// WithAttemptsByError 方法设置 Config 的错误重试次数并返回 Config 实例
// The WithAttemptsByError method sets the number of error retries of the Config and returns the Config instance
func (c *Config) WithAttemptsByError(attemptsByError map[error]uint64) *Config {
	c.attemptsByError = attemptsByError
	return c
}

// WithFactor 方法设置 Config 的因子并返回 Config 实例
// The WithFactor method sets the factor of the Config and returns the Config instance
func (c *Config) WithFactor(factor float64) *Config {
	c.factor = factor
	return c
}

// WithInitDelay 方法设置 Config 的初始延迟时间并返回 Config 实例
// The WithInitDelay method sets the initial delay time of the Config and returns the Config instance
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
	if conf == nil {
		conf = NewConfig()
	} else {
		if conf.ctx == nil {
			conf.ctx = context.Background()
		}
		if conf.callback == nil {
			conf.callback = NewEmptyCallback()
		}
		if conf.attempts <= 0 || conf.attempts >= math.MaxUint16 {
			conf.attempts = defaultAttempts
		}
		if conf.attemptsByError == nil {
			conf.attemptsByError = make(map[error]uint64)
		}
		if conf.factor < 0 {
			conf.factor = defaultFactor
		}
		if conf.delay <= 0 {
			conf.delay = defaultDelay
		}
		if conf.jitter < 0 {
			conf.jitter = defaultJitter
		}
		if conf.retryIfFunc == nil {
			conf.retryIfFunc = defaultRetryIfFunc
		}
		if conf.backoffFunc == nil {
			conf.backoffFunc = defaultBackoffFunc
		}
	}

	return conf
}

// DefaultConfig 函数返回一个新的默认配置的 Config 实例
// The DefaultConfig function returns a new Config instance with the default configuration
func DefaultConfig() *Config {
	return NewConfig()
}

// FixConfig 函数返回一个新的固定退避时间的 Config 实例
// The FixConfig function returns a new Config instance with a fixed backoff time
func FixConfig() *Config {
	return NewConfig().WithBackOffFunc(FixBackOff).WithFactor(0).WithJitter(0)
}
