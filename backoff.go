package retry

import (
	"math"
	"math/rand"
	"time"
)

// 定义基础时间单位
// Define the base time unit
const baseTimeDuration = 100 * time.Millisecond

// BackoffFunc 类型定义了一个接受整数并返回时间间隔的函数类型
// The BackoffFunc type defines a function type that accepts an integer and returns a time interval
type BackoffFunc = func(int64) time.Duration

// FixBackOff 函数返回一个固定的时间间隔
// The FixBackOff function returns a fixed time interval
func FixBackOff(delay int64) time.Duration {
	return time.Duration(delay) * baseTimeDuration
}

// RandomBackOff 函数返回一个随机的时间间隔
// The RandomBackOff function returns a random time interval
func RandomBackOff(delay int64) time.Duration {
	return time.Duration(rand.Int63n(delay)) * baseTimeDuration
}

// ExponentialBackOff 函数返回一个指数增长的时间间隔
// The ExponentialBackOff function returns an exponentially increasing time interval
func ExponentialBackOff(delay int64) time.Duration {
	return time.Duration(int64(math.Exp2(float64(delay)))) * baseTimeDuration
}

// CombineBackOffs 函数组合多个退避函数，并返回一个新的退避函数
// The CombineBackOffs function combines multiple backoff functions and returns a new backoff function
func CombineBackOffs(backoffs ...BackoffFunc) BackoffFunc {
	return func(n int64) time.Duration {
		var delay time.Duration
		// 对每个退避函数进行调用，并累加它们的结果
		// Call each backoff function and accumulate their results
		for _, backoff := range backoffs {
			delay += backoff(n)
		}

		// 如果计算出的延迟时间小于等于0，则返回默认的延迟时间
		// If the calculated delay time is less than or equal to 0, return the default delay time
		if delay <= 0 {
			return defaultDelay
		}

		// 返回计算出的延迟时间
		// Return the calculated delay time
		return delay
	}
}
