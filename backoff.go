package retry

import (
	"math"
	"math/rand"
	"time"
)

// 基准时间间隔
// Base time interval
const baseTimeDuration = 100 * time.Millisecond

// 用于计算重试间隔
// Used to calculate the retry interval.
type BackoffFunc = func(int64) time.Duration

// FixBackOff 方法用于固定间隔重试
// The FixBackOff method is used to retry at a fixed interval.
func FixBackOff(delay int64) time.Duration {
	return time.Duration(delay) * baseTimeDuration
}

// RandomBackOff 方法用于随机间隔重试
// The RandomBackOff method is used to retry at a random interval.
func RandomBackOff(delay int64) time.Duration {
	return time.Duration(rand.Int63n(delay)) * baseTimeDuration
}

// ExponentialBackOff 方法用于指数间隔重试
// The ExponentialBackOff method is used to retry at an exponential interval.
func ExponentialBackOff(delay int64) time.Duration {
	return time.Duration(int64(math.Exp2(float64(delay)))) * baseTimeDuration
}

// CombineBackOffs 方法用于组合多个重试间隔
// The CombineBackOffs method is used to combine multiple retry intervals.
func CombineBackOffs(backoffs ...BackoffFunc) BackoffFunc {
	return func(n int64) time.Duration {
		var delay time.Duration
		// 依次计算每个重试间隔，并行性相加
		// Calculate each retry interval in turn and add it in parallel.
		for _, backoff := range backoffs {
			delay += backoff(n)
		}

		// 如果重试间隔小于等于 0，则返回默认重试间隔
		// If the retry interval is less than or equal to 0, the default retry interval is returned.
		if delay <= 0 {
			return defaultDelay
		}

		// 返回重试间隔
		// Return the retry interval.
		return delay
	}
}
