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
			expected: (1 << 3) * baseInterval,
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
func TestExponentialBackoffOverflowClamp(t *testing.T) {
	// 独立推导的钳制上界：2^36 个 100ms 基础时间单位
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
				_ = RandomBackoff(5)
				_ = ExponentialBackoff(3)
				combined := CombineBackoffs(FixedBackoff, ExponentialBackoff)
				_ = combined(3)
			}
		}()
	}

	wg.Wait()
}

// TestFixedBackoffOverflow 验证大 interval 值被钳制而不溢出：
// math.MaxInt64 直接乘 baseInterval 会回绕为负值，钳制后应返回正值。
func TestFixedBackoffOverflow(t *testing.T) {
	// 独立推导的钳制期望值：maxSafeInterval * baseInterval
	expected := time.Duration(maxSafeInterval) * baseInterval

	tests := []struct {
		name  string
		input int64
	}{
		{
			name:  "max int64",
			input: math.MaxInt64,
		},
		{
			name:  "large overflow value",
			input: math.MaxInt64 / 10,
		},
		{
			name:  "just above safe limit",
			input: math.MaxInt64/int64(baseInterval) + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixedBackoff(tt.input)
			assert.Positive(t, int64(result), "FixedBackoff(%d) 必须为正值，不得溢出回绕", tt.input)
			assert.Equal(t, expected, result, "FixedBackoff(%d) 必须钳制为 maxSafeInterval * baseInterval", tt.input)
		})
	}
}

// TestRandomBackoffOverflow 验证大 maxInterval 值被钳制而不溢出：
// 多次调用结果均应为非负值，不得出现溢出回绕。
func TestRandomBackoffOverflow(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{
			name:  "max int64",
			input: math.MaxInt64,
		},
		{
			name:  "large overflow value",
			input: math.MaxInt64 / 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 100 {
				result := RandomBackoff(tt.input)
				assert.GreaterOrEqual(t, int64(result), int64(0), "RandomBackoff(%d) 必须为非负值，不得溢出回绕", tt.input)
			}
		})
	}
}

// TestCombineBackoffsNilElement 验证含 nil 元素的 backoffs 切片不 panic：
// 跳过 nil 后应正常累加其余元素。
func TestCombineBackoffsNilElement(t *testing.T) {
	tests := []struct {
		name     string
		backoffs []BackoffFunc
		input    int64
		expected time.Duration
	}{
		{
			name:     "nil at start",
			backoffs: []BackoffFunc{nil, FixedBackoff},
			input:    3,
			expected: 3 * baseInterval,
		},
		{
			name:     "nil in middle",
			backoffs: []BackoffFunc{FixedBackoff, nil, ExponentialBackoff},
			input:    3,
			expected: time.Duration(3+int64(math.Exp2(3))) * baseInterval,
		},
		{
			name:     "nil at end",
			backoffs: []BackoffFunc{FixedBackoff, nil},
			input:    3,
			expected: 3 * baseInterval,
		},
		{
			name:     "all nil",
			backoffs: []BackoffFunc{nil, nil, nil},
			input:    3,
			expected: defaultDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combined := CombineBackoffs(tt.backoffs...)
			assert.NotPanics(t, func() {
				result := combined(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		})
	}
}

// TestCombineBackoffsOverflow 验证 3+ 个大退避值累加不溢出：
// 饱和加法应将结果钳制为 math.MaxInt64，而非回绕为小正值。
func TestCombineBackoffsOverflow(t *testing.T) {
	// 构造返回极大值的 backoff 函数
	largeBackoff := func(_ int64) time.Duration {
		return time.Duration(math.MaxInt64 / 2)
	}

	tests := []struct {
		name     string
		backoffs []BackoffFunc
		input    int64
		expected time.Duration
	}{
		{
			name:     "three large backoffs",
			backoffs: []BackoffFunc{largeBackoff, largeBackoff, largeBackoff},
			input:    1,
			expected: time.Duration(math.MaxInt64),
		},
		{
			name:     "five large backoffs",
			backoffs: []BackoffFunc{largeBackoff, largeBackoff, largeBackoff, largeBackoff, largeBackoff},
			input:    1,
			expected: time.Duration(math.MaxInt64),
		},
		{
			name: "mixed with fixed overflow",
			backoffs: []BackoffFunc{
				func(_ int64) time.Duration { return time.Duration(math.MaxInt64 - 1000) },
				func(_ int64) time.Duration { return time.Duration(2000) },
			},
			input:    1,
			expected: time.Duration(math.MaxInt64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combined := CombineBackoffs(tt.backoffs...)
			result := combined(tt.input)
			assert.Positive(t, int64(result), "CombineBackoffs 累加结果必须为正值，不得溢出回绕")
			assert.Equal(t, tt.expected, result, "CombineBackoffs 饱和加法必须将结果钳制为 math.MaxInt64")
		})
	}
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
