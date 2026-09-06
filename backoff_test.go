package retry

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFixedBackoff(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected time.Duration
	}{
		{
			name:     "normal case",
			input:    3,
			expected: 3 * baseInterval,
		},
		{
			name:     "zero input",
			input:    0,
			expected: defaultDelay,
		},
		{
			name:     "negative input",
			input:    -1,
			expected: defaultDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixedBackoff(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandomBackoff(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{
			name:  "normal case",
			input: 5,
		},
		{
			name:  "zero input",
			input: 0,
		},
		{
			name:  "negative input",
			input: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandomBackoff(tt.input)
			if tt.input <= 0 {
				assert.Equal(t, defaultDelay, result)
			} else {
				assert.LessOrEqual(t, result, time.Duration(tt.input)*baseInterval)
				assert.GreaterOrEqual(t, result, time.Duration(0))
			}
		})
	}
}

func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected time.Duration
	}{
		{
			name:     "normal case",
			input:    3,
			expected: time.Duration(int64(math.Exp2(3))) * baseInterval,
		},
		{
			name:     "zero input",
			input:    0,
			expected: defaultDelay,
		},
		{
			name:     "negative input",
			input:    -1,
			expected: defaultDelay,
		},
		{
			// 期望值独立推导：钳制后应为 2^36 * 100ms，不使用被测代码同款 Exp2 表达式计算
			// Expected value derived independently: clamped result must be 2^36 * 100ms, not computed with the same Exp2 expression as the code under test
			name:     "max exponential",
			input:    maxExponent + 1,
			expected: (1 << 36) * baseInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExponentialBackoff(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExponentialBackoffOverflowClamp 验证大指数输入被钳制为不溢出的正值：
// 2^36 * 100ms = 6871947673600000000ns < math.MaxInt64，而 2^37 * 100ms 会回绕为负值。
// TestExponentialBackoffOverflowClamp verifies large power inputs are clamped to a non-overflowing positive value:
// 2^36 * 100ms = 6871947673600000000ns < math.MaxInt64, while 2^37 * 100ms wraps around to a negative value.
func TestExponentialBackoffOverflowClamp(t *testing.T) {
	// 独立推导的钳制上界：2^36 个 100ms 基础时间单位
	// Independently derived clamp bound: 2^36 units of the 100ms base interval
	const clamped = (1 << 36) * baseInterval

	for _, power := range []int64{37, 62, 100} {
		result := ExponentialBackoff(power)
		assert.Positive(t, int64(result), "ExponentialBackoff(%d) 必须为正值，不得溢出回绕", power)
		assert.Equal(t, time.Duration(clamped), result, "ExponentialBackoff(%d) 必须钳制为 2^36 * 100ms", power)
	}
}

func TestCombineBackoffs(t *testing.T) {
	tests := []struct {
		name     string
		backoffs []BackoffFunc
		input    int64
		expected time.Duration
	}{
		{
			name:     "empty backoffs",
			backoffs: []BackoffFunc{},
			input:    3,
			expected: FixedBackoff(3),
		},
		{
			name:     "single backoff",
			backoffs: []BackoffFunc{FixedBackoff},
			input:    3,
			expected: 3 * baseInterval,
		},
		{
			name:     "multiple backoffs",
			backoffs: []BackoffFunc{FixedBackoff, ExponentialBackoff},
			input:    3,
			expected: time.Duration(3+int64(math.Exp2(3))) * baseInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combined := CombineBackoffs(tt.backoffs...)
			result := combined(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConcurrentBackoffs(t *testing.T) {
	const (
		goroutines = 100
		iterations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				// Test all backoff functions concurrently
				_ = RandomBackoff(5)
				_ = ExponentialBackoff(3)
				combined := CombineBackoffs(FixedBackoff, ExponentialBackoff)
				_ = combined(3)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkFixedBackoff(b *testing.B) {
	for range b.N {
		_ = FixedBackoff(3)
	}
}

func BenchmarkRandomBackoff(b *testing.B) {
	for range b.N {
		_ = RandomBackoff(3)
	}
}

func BenchmarkExponentialBackoff(b *testing.B) {
	for range b.N {
		_ = ExponentialBackoff(3)
	}
}

func BenchmarkCombinedBackoffs(b *testing.B) {
	combined := CombineBackoffs(FixedBackoff, ExponentialBackoff)
	b.ResetTimer()
	for range b.N {
		_ = combined(3)
	}
}
