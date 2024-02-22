package retry

import (
	"context"
	"math"
	"time"
)

const (
	// 默认重试次数
	// Default number of retries
	defaultAttempts = 3

	// 默认延迟时间
	// Default delay time
	defaultDelayNum = 5

	// 默认重试间隔
	// Default retry interval
	defaultDelay = defaultDelayNum * time.Millisecond * 100

	// 默认抖动
	// Default jitter
	defaultJitter = 3.0

	// 默认因子
	// Default factor
	defaultFactor = 1.0
)

var (
	// 默认重试条件
	// Default retry condition
	defaultRetryIfFunc = func(error) bool { return true }

	// 默认间隔策略
	// Default interval strategy
	defaultBackoffFunc = func(n int64) time.Duration {
		return CombineBackOffs(ExponentialBackOff, RandomBackOff)(n) // 默认间隔 (指数退避 + 随机退避) 策略
	}
)

// Callback 方法用于定义重试回调函数
// The Callback method is used to define the retry callback function.
type Callback interface {
	OnRetry(count int64, delay time.Duration, err error)
}

// emptyCallback 用于实现 Callback 接口
// emptyCallback is used to implement the Callback interface.
type emptyCallback struct{}

// OnRetry 方法用于实现 Callback 接口
// The OnRetry method is used to implement the Callback interface.
func (cb *emptyCallback) OnRetry(count int64, delay time.Duration, err error) {}

// NewEmptyCallback 方法用于创建一个空的回调函数
// The NewEmptyCallback method is used to create an empty callback function.
func NewEmptyCallback() Callback {
	return &emptyCallback{}
}

// 用于判断是否重试
// Used to determine whether to retry.
type RetryIfFunc = func(error) bool

// Config 定义了重试配置
// Config defines the retry configuration.
type Config struct {
	ctx             context.Context
	callback        Callback
	attempts        uint64
	attemptsByError map[error]uint64
	factor          float64
	jitter          float64
	delay           time.Duration
	retryIfFunc     RetryIfFunc
	backoffFunc     BackoffFunc
	detail          bool
}

// NewConfig 方法用于创建一个新的配置
// The NewConfig method is used to create a new configuration.
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

// WithContext 方法用于设置上下文
// The WithContext method is used to set the context.
func (c *Config) WithContext(ctx context.Context) *Config {
	c.ctx = ctx
	return c
}

// WithCallback 方法用于设置回调函数
// The WithCallback method is used to set the callback function.
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

// WithAttempts 方法用于设置重试次数
// The WithAttempts method is used to set the number of retries.
func (c *Config) WithAttempts(attempts uint64) *Config {
	c.attempts = attempts
	return c
}

// WithAttemptsByError 方法用于设置指定错误的重试次数，所有错误的充数次数的总和应该小于 WithAttempts 方法设置的重试次数
// The WithAttemptsByError method is used to set the number of retries for the specified error.
// The total number of retries for all errors should be less than the number of retries set by the WithAttempts method.
func (c *Config) WithAttemptsByError(attemptsByError map[error]uint64) *Config {
	c.attemptsByError = attemptsByError
	return c
}

// WithFactor 方法用于设置重试因子
// The WithFactor method is used to set the retry factor.
func (c *Config) WithFactor(factor float64) *Config {
	c.factor = factor
	return c
}

// WithInitDelay 方法用于设置初始重试延迟
// The WithInitDelay method is used to set the initial retry delay.
func (c *Config) WithInitDelay(delay time.Duration) *Config {
	c.delay = delay
	return c
}

// WithJitter 方法用于设置重试抖动
// The WithJitter method is used to set the retry jitter.
func (c *Config) WithJitter(jitter float64) *Config {
	c.jitter = jitter
	return c
}

// WithRetryIfFunc 方法用于设置重试条件
// The WithRetryIfFunc method is used to set the retry condition.
func (c *Config) WithRetryIfFunc(retryIf RetryIfFunc) *Config {
	c.retryIfFunc = retryIf
	return c
}

// WithBackOffFunc 方法用于设置重试间隔
// The WithBackOffFunc method is used to set the retry interval.
func (c *Config) WithBackOffFunc(backoff BackoffFunc) *Config {
	c.backoffFunc = backoff
	return c
}

// WithDetail 方法用于设置是否记录重试过程中的错误
// The WithDetail method is used to set whether to record errors during the retry process.
func (c *Config) WithDetail(detail bool) *Config {
	c.detail = detail
	return c
}

// isConfigValid 方法用于检查配置是否合法
// The isConfigValid method is used to check whether the configuration is valid.
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

// DefaultConfig 方法用于生成默认配置
// The DefaultConfig method is used to generate the default configuration.
func DefaultConfig() *Config {
	return NewConfig()
}

// FixConfig 方法用于生成没有抖动和因子的配置
// The FixConfig method is used to generate a configuration without jitter and factor.
func FixConfig() *Config {
	return NewConfig().WithBackOffFunc(FixBackOff).WithFactor(0).WithJitter(0)
}
