package retry

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	// 基础时间单位为100毫秒
	// Base time unit is 100 milliseconds
	baseInterval = 100 * time.Millisecond

	// 防止 time.Duration 溢出的最大指数值：
	// 2^36 * 100ms = 6871947673600000000ns < math.MaxInt64；2^37 * 100ms 乘法回绕为负值。
	// 62 只能保证 Exp2 结果本身可转 int64，不能保证乘 baseInterval 后不溢出。
	// Maximum exponent to prevent time.Duration overflow:
	// 2^36 * 100ms = 6871947673600000000ns < math.MaxInt64; 2^37 * 100ms wraps around to a negative value.
	// 62 would only guarantee the Exp2 result fits in int64, not the multiplication by baseInterval.
	maxExponent = 36
)

var (
	// 使用独立的随机数生成器，避免全局锁竞争
	// Use a separate random number generator to avoid global lock contention
	randGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	randMu  sync.Mutex
)

// BackoffFunc 定义了退避策略函数的类型：入参为当前已完成的执行次数 count（从 1 开始），返回本次重试前的退避时长
// BackoffFunc defines the type for backoff strategy functions: the parameter is the current number of
// completed executions count (starting from 1), and it returns the backoff duration before the next retry
type BackoffFunc = func(int64) time.Duration

// FixedBackoff 返回固定时间间隔的退避策略：interval * 100ms（基础时间单位）；interval <= 0 时返回默认延迟 500ms
// FixedBackoff returns a fixed-interval backoff strategy: interval * 100ms (the base time unit);
// if interval <= 0, it returns the default delay of 500ms
func FixedBackoff(interval int64) time.Duration {
	if interval <= 0 {
		return defaultDelay
	}
	return time.Duration(interval) * baseInterval
}

// RandomBackoff 返回随机时间间隔的退避策略：[0, maxInterval) 内的随机整数 * 100ms；maxInterval <= 0 时返回默认延迟 500ms
// RandomBackoff returns a random-interval backoff strategy: a random integer in [0, maxInterval) * 100ms;
// if maxInterval <= 0, it returns the default delay of 500ms
func RandomBackoff(maxInterval int64) time.Duration {
	if maxInterval <= 0 {
		return defaultDelay
	}

	randMu.Lock()
	interval := randGen.Int63n(maxInterval)
	randMu.Unlock()

	return time.Duration(interval) * baseInterval
}

// ExponentialBackoff 返回指数增长的退避策略：2^power * 100ms。
// power 超过上限 36 时被钳制为 36（power > 36 会使乘以 100ms 后溢出 int64）；power <= 0 时返回默认延迟 500ms
// ExponentialBackoff returns an exponential backoff strategy: 2^power * 100ms.
// power is clamped to the maximum of 36 (power > 36 would overflow int64 after multiplying by 100ms);
// if power <= 0, it returns the default delay of 500ms
func ExponentialBackoff(power int64) time.Duration {
	if power <= 0 {
		return defaultDelay
	}

	// 限制最大指数以防止溢出
	// Limit maximum exponent to prevent overflow
	if power > maxExponent {
		power = maxExponent
	}

	return time.Duration(int64(math.Exp2(float64(power)))) * baseInterval
}

// CombineBackoffs 将多个退避策略组合成一个：对同一入参 count 逐项求和。
// backoffs 为空时返回 FixedBackoff；求和结果 <= 0 时返回默认延迟 500ms
// CombineBackoffs combines multiple backoff strategies into one by summing their results for the same
// count. If backoffs is empty, it returns FixedBackoff; if the sum is <= 0, it returns the default delay of 500ms
func CombineBackoffs(backoffs ...BackoffFunc) BackoffFunc {
	if len(backoffs) == 0 {
		return FixedBackoff
	}

	return func(n int64) time.Duration {
		var totalDelay time.Duration
		for _, backoff := range backoffs {
			totalDelay += backoff(n)
		}

		if totalDelay <= 0 {
			return defaultDelay
		}
		return totalDelay
	}
}
