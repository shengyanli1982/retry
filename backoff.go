package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

const (
	// 基础时间单位为100毫秒
	baseInterval = 100 * time.Millisecond

	// 防止 time.Duration 溢出的最大指数值：
	// 2^36 * 100ms = 6871947673600000000ns < math.MaxInt64；2^37 * 100ms 乘法回绕为负值。
	// 62 只能保证移位结果 1<<62 本身不溢出 int64，不能保证乘 baseInterval 后不溢出。
	maxExponent = 36

	// 防止 time.Duration 乘法溢出的最大安全 interval 值：
	// interval * baseInterval 不得超过 math.MaxInt64
	maxSafeInterval = math.MaxInt64 / int64(baseInterval)
)

// BackoffFunc 定义了退避策略函数的类型：入参为当前已完成的执行次数 count（从 1 开始），返回本次重试前的退避时长
type BackoffFunc = func(int64) time.Duration

// FixedBackoff 返回固定时间间隔的退避策略：interval * 100ms（基础时间单位）；interval <= 0 时返回默认延迟 500ms
func FixedBackoff(interval int64) time.Duration {
	if interval <= 0 {
		return defaultDelay
	}

	// 钳制 interval 以防止 time.Duration 乘法溢出：interval * baseInterval 不得超过 math.MaxInt64
	if interval > maxSafeInterval {
		interval = maxSafeInterval
	}

	return time.Duration(interval) * baseInterval
}

// RandomBackoff 返回随机时间间隔的退避策略：[0, maxInterval) 内的随机整数 * 100ms；maxInterval <= 0 时返回默认延迟 500ms
func RandomBackoff(maxInterval int64) time.Duration {
	if maxInterval <= 0 {
		return defaultDelay
	}

	// 钳制 maxInterval 以防止 time.Duration 乘法溢出：rand.Int64N 返回值 * baseInterval 不得超过 math.MaxInt64
	if maxInterval > maxSafeInterval {
		maxInterval = maxSafeInterval
	}

	// rand/v2 顶层函数 goroutine 安全且 per-M 无锁；maxInterval <= 0 已被前置守卫拦截，Int64N 不会收到非正参数
	interval := rand.Int64N(maxInterval)

	return time.Duration(interval) * baseInterval
}

// ExponentialBackoff 返回指数增长的退避策略：2^power * 100ms。
// power 超过上限 36 时被钳制为 36（power > 36 会使乘以 100ms 后溢出 int64）；power <= 0 时返回默认延迟 500ms
func ExponentialBackoff(power int64) time.Duration {
	if power <= 0 {
		return defaultDelay
	}

	// 限制最大指数以防止溢出
	if power > maxExponent {
		power = maxExponent
	}

	// 整数移位精确计算 2^power，免于 float 往返；power 已钳制在 [1, maxExponent]，
	// 移位与乘 baseInterval 均不溢出，数值与原 Exp2 实现逐一相同
	return time.Duration(int64(1)<<power) * baseInterval
}

// CombineBackoffs 将多个退避策略组合成一个：对同一入参 count 逐项求和。
// backoffs 为空时返回 FixedBackoff；求和结果 <= 0 时返回默认延迟 500ms
func CombineBackoffs(backoffs ...BackoffFunc) BackoffFunc {
	if len(backoffs) == 0 {
		return FixedBackoff
	}

	return func(n int64) time.Duration {
		var totalDelay time.Duration
		for _, backoff := range backoffs {
			// 跳过 nil 元素以防止 panic
			if backoff == nil {
				continue
			}

			d := backoff(n)

			// 饱和加法：累加前检查是否溢出，溢出则钳制为 math.MaxInt64
			if d > math.MaxInt64-totalDelay {
				totalDelay = math.MaxInt64
			} else {
				totalDelay += d
			}
		}

		if totalDelay <= 0 {
			return defaultDelay
		}
		return totalDelay
	}
}
