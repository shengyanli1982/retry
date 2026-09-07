package retry

import (
	"context"
	"maps"
	"math"
	"time"
)

// 定义默认的重试次数、延迟时间、抖动和因子
const (
	defaultAttempts = 3                                        // 默认的重试次数为3次
	defaultDelayNum = 5                                        // 默认的延迟基数为5（对应5个100毫秒基础时间单位）
	defaultDelay    = defaultDelayNum * time.Millisecond * 100 // 默认的延迟时间：5 * 1ms * 100 = 500毫秒
	defaultJitter   = 3.0                                      // 默认的抖动为3.0
	defaultFactor   = 1.0                                      // 默认的因子为1.0
)

// 定义默认的重试条件函数和退避函数
var (
	// defaultRetryIfFunc 是默认的重试条件函数，对所有错误都进行重试
	defaultRetryIfFunc = func(error) bool { return true }

	// defaultBackoffFunc 是默认的退避函数，使用指数退避和随机退避的组合。
	// 包级一次性构造：闭包无状态（随机数由 math/rand/v2 顶层函数提供，goroutine 安全且无锁），并发安全，避免每次调用重建组合闭包。
	defaultBackoffFunc = CombineBackoffs(ExponentialBackoff, RandomBackoff)
)

// 定义一个空的回调结构体
type emptyCallback struct{}

// OnRetry 方法在每次重试时调用，但不执行任何操作
func (cb *emptyCallback) OnRetry(count int64, delay time.Duration, err error) {}

// NewEmptyCallback 函数返回一个新的空回调实例
func NewEmptyCallback() Callback {
	return &emptyCallback{}
}

// RetryIfFunc 类型定义了一个接受错误并返回布尔值的函数类型
type RetryIfFunc = func(error) bool

// Config 结构体定义了重试的配置
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
func NewConfig() *Config {
	return &Config{
		ctx:         context.Background(),
		callback:    NewEmptyCallback(),
		attempts:    defaultAttempts,
		factor:      defaultFactor,
		delay:       defaultDelay,
		jitter:      defaultJitter,
		retryIfFunc: defaultRetryIfFunc,
		backoffFunc: defaultBackoffFunc,
		detail:      false,
	}
}

// WithContext 方法设置 Config 的上下文并返回 Config 实例
func (c *Config) WithContext(ctx context.Context) *Config {
	c.ctx = ctx
	return c
}

// WithCallback 方法设置 Config 的回调函数并返回 Config 实例
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

// WithAttempts 方法设置 Config 的重试次数并返回 Config 实例。
// 取值在 New 时归一化：0 → 默认 3；>= 65535 → 钳制为 65534（见 isConfigValid）。
func (c *Config) WithAttempts(attempts uint64) *Config {
	c.attempts = attempts
	return c
}

// WithAttemptsByError 方法设置 Config 的错误重试次数并返回 Config 实例。
// 入口拷贝用户 map，库持有私有副本：用户后续对原 map 的写入不影响重试预算，
// 也避免重试执行期间与用户并发写相遇导致 fatal error: concurrent map read and map write。
func (c *Config) WithAttemptsByError(attemptsByError map[error]uint64) *Config {
	// maps.Clone 拷贝用户 map；maps.Clone(nil) 返回 nil。
	// nil 语义安全：TryOnConflict 以 len > 0 守卫决定是否 Clone 本地副本，且对副本仅有 ok 守卫下的读写（nil map 读返回零值 + ok=false）
	c.attemptsByError = maps.Clone(attemptsByError)
	return c
}

// WithFactor 方法设置 Config 的因子并返回 Config 实例
func (c *Config) WithFactor(factor float64) *Config {
	c.factor = factor
	return c
}

// WithInitDelay 方法设置 Config 的基础延迟（每次重试线性延迟项的乘数基数，见 retry.go 的退避计算）并返回 Config 实例
func (c *Config) WithInitDelay(delay time.Duration) *Config {
	c.delay = delay
	return c
}

// WithJitter 方法设置 Config 的抖动并返回 Config 实例
func (c *Config) WithJitter(jitter float64) *Config {
	c.jitter = jitter
	return c
}

// WithRetryIfFunc 方法设置 Config 的重试条件函数并返回 Config 实例
func (c *Config) WithRetryIfFunc(retryIf RetryIfFunc) *Config {
	c.retryIfFunc = retryIf
	return c
}

// WithBackOffFunc 方法设置 Config 的退避函数并返回 Config 实例
func (c *Config) WithBackOffFunc(backoff BackoffFunc) *Config {
	c.backoffFunc = backoff
	return c
}

// WithDetail 方法设置 Config 的详细错误信息显示选项并返回 Config 实例
func (c *Config) WithDetail(detail bool) *Config {
	c.detail = detail
	return c
}

// isConfigValid 函数检查 Config 是否有效，如果无效则使用默认值
func isConfigValid(conf *Config) *Config {
	// 如果 conf 为 nil，则创建一个新的 Config 实例
	if conf == nil {
		conf = NewConfig()
	} else {
		// 如果 conf.ctx 为 nil，则设置为默认的上下文
		if conf.ctx == nil {
			conf.ctx = context.Background()
		}

		// 如果 conf.callback 为 nil，则设置为默认的回调函数
		if conf.callback == nil {
			conf.callback = NewEmptyCallback()
		}

		// 如果 conf.attempts 为 0，则重置为默认的重试次数；如果达到或超过 math.MaxUint16（65535），则钳制为 math.MaxUint16-1（65534）。
		// 上限钳制而非重置的原因：65534 是可表示的最大合法重试预算，把巨大值静默重置为默认 3 会造成"要求更多却得到更少"的反直觉语义。
		if conf.attempts <= 0 {
			conf.attempts = defaultAttempts
		} else if conf.attempts >= math.MaxUint16 {
			conf.attempts = math.MaxUint16 - 1
		}

		// 如果 conf.factor 小于 0，则设置为默认的退避因子
		if conf.factor < 0 {
			conf.factor = defaultFactor
		}

		// 如果 conf.delay 小于等于 0，则设置为默认的延迟时间
		if conf.delay <= 0 {
			conf.delay = defaultDelay
		}

		// 如果 conf.jitter 小于 0，则设置为默认的抖动
		if conf.jitter < 0 {
			conf.jitter = defaultJitter
		}

		// 如果 conf.retryIfFunc 为 nil，则设置为默认的重试条件函数
		if conf.retryIfFunc == nil {
			conf.retryIfFunc = defaultRetryIfFunc
		}

		// 如果 conf.backoffFunc 为 nil，则设置为默认的退避函数
		if conf.backoffFunc == nil {
			conf.backoffFunc = defaultBackoffFunc
		}
	}

	// 返回检查并修正后的 Config 实例
	return conf
}

// DefaultConfig 函数返回一个新的默认配置的 Config 实例
func DefaultConfig() *Config {
	return NewConfig()
}

// FixConfig 函数返回一个新的固定退避时间的 Config 实例：每次重试固定间隔 500ms
// （backoffFunc 恒返回 defaultDelay，factor=0、jitter=0 使线性延迟项为 0）
func FixConfig() *Config {
	return NewConfig().WithBackOffFunc(func(_ int64) time.Duration {
		return defaultDelay
	}).WithFactor(0).WithJitter(0)
}
