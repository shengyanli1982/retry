package retry

import (
	"context"
	"time"
)

const (
	defaultAttempts = 3
	defaultDelayNum = 5
	defaultDelay    = defaultDelayNum * time.Millisecond * 100
	defaultJitter   = 3.0
	defaultFactor   = 1.0
)

var (
	defaultRetryIf = func(error) bool { return true }
	defaultBackoff = func(n int64) time.Duration {
		return CombineBackOffs(ExponentialBackOff, RandomBackOff)(n)
	}
)

// Callback 方法用于定义重试回调函数
// The Callback method is used to define the retry callback function.
type Callback interface {
	OnRetry(count int64, delay time.Duration, err error)
}

type emptyCallback struct{}

func (cb *emptyCallback) OnRetry(count int64, delay time.Duration, err error) {}

// 用于判断是否重试
// Used to determine whether to retry.
type RetryIfFunc func(error) bool

type Config struct {
	ctx             context.Context
	cb              Callback
	attempts        uint64
	attemptsByError map[error]uint64
	factor          float64
	jitter          float64
	delay           time.Duration
	retryIf         RetryIfFunc
	backoff         BackoffFunc
	detail          bool
}

// NewConfig 方法用于创建一个新的配置
// The NewConfig method is used to create a new configuration.
func NewConfig() *Config {
	return &Config{
		ctx:             context.Background(),
		cb:              &emptyCallback{},
		attempts:        defaultAttempts,
		attemptsByError: make(map[error]uint64),
		factor:          defaultFactor,
		delay:           defaultDelay,
		jitter:          defaultJitter,
		retryIf:         defaultRetryIf,
		backoff:         defaultBackoff,
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
	c.cb = cb
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

// WithDelay 方法用于设置重试延迟
// The WithDelay method is used to set the retry delay.
func (c *Config) WithDelay(delay time.Duration) *Config {
	c.delay = delay
	return c
}

// WithJitter 方法用于设置重试抖动
// The WithJitter method is used to set the retry jitter.
func (c *Config) WithJitter(jitter float64) *Config {
	c.jitter = jitter
	return c
}

// WithRetryIf 方法用于设置重试条件
// The WithRetryIf method is used to set the retry condition.
func (c *Config) WithRetryIf(retryIf RetryIfFunc) *Config {
	c.retryIf = retryIf
	return c
}

// WithBackoff 方法用于设置重试间隔
// The WithBackoff method is used to set the retry interval.
func (c *Config) WithBackoff(backoff BackoffFunc) *Config {
	c.backoff = backoff
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
		if conf.cb == nil {
			conf.cb = &emptyCallback{}
		}
		if conf.attempts <= 0 {
			conf.attempts = defaultAttempts
		}
		if conf.attemptsByError == nil {
			conf.attemptsByError = make(map[error]uint64)
		}
		if conf.factor <= 0 {
			conf.factor = defaultFactor
		}
		if conf.delay <= 0 {
			conf.delay = defaultDelay
		}
		if conf.jitter < 0 {
			conf.jitter = defaultJitter
		}
		if conf.retryIf == nil {
			conf.retryIf = defaultRetryIf
		}
		if conf.backoff == nil {
			conf.backoff = defaultBackoff
		}
	}

	return conf
}
