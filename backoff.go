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

	// 防止 time.Duration 溢出的最大指数值
	// Maximum exponent to prevent time.Duration overflow
	maxExponent = 62
)

var (
	// 使用独立的随机数生成器，避免全局锁竞争
	// Use a separate random number generator to avoid global lock contention
	randGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	randMu  sync.Mutex
)

// BackoffFunc 定义了退避策略函数的类型
// BackoffFunc defines the type for backoff strategy functions
type BackoffFunc = func(int64) time.Duration

// FixedBackoff 返回固定时间间隔的退避策略
// FixedBackoff returns a fixed-interval backoff strategy
func FixedBackoff(interval int64) time.Duration {
	if interval <= 0 {
		return defaultDelay
	}
	return time.Duration(interval) * baseInterval
}

// RandomBackoff 返回随机时间间隔的退避策略
// RandomBackoff returns a random-interval backoff strategy
func RandomBackoff(maxInterval int64) time.Duration {
	if maxInterval <= 0 {
		return defaultDelay
	}

	randMu.Lock()
	interval := randGen.Int63n(maxInterval)
	randMu.Unlock()

	return time.Duration(interval) * baseInterval
}

// ExponentialBackoff 返回指数增长的退避策略
// ExponentialBackoff returns an exponential backoff strategy
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

// CombineBackoffs 将多个退避策略组合成一个
// CombineBackoffs combines multiple backoff strategies into one
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
