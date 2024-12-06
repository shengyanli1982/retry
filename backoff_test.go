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
			name:     "max exponential",
			input:    maxExponent + 1,
			expected: time.Duration(int64(math.Exp2(float64(maxExponent)))) * baseInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExponentialBackoff(tt.input)
			assert.Equal(t, tt.expected, result)
		})
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

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
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
	for i := 0; i < b.N; i++ {
		_ = FixedBackoff(3)
	}
}

func BenchmarkRandomBackoff(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = RandomBackoff(3)
	}
}

func BenchmarkExponentialBackoff(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ExponentialBackoff(3)
	}
}

func BenchmarkCombinedBackoffs(b *testing.B) {
	combined := CombineBackoffs(FixedBackoff, ExponentialBackoff)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = combined(3)
	}
}
