package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type callback struct{}

func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

func TestRetry_Do(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	cfg := NewConfig().WithAttemptsByError(m).WithDetail(true)

	count := 0
	testFunc := func() (any, error) {
		if count > 0 {
			return "lee", nil
		} else {
			count++
			return nil, e
		}
	}

	result := Do(testFunc, cfg)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e})
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(2))

}

func TestRetry_DoWithDefault(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	count := 0
	testFunc := func() (any, error) {
		if count > 0 {
			return "lee", nil
		} else {
			count++
			return nil, e
		}
	}

	result := DoWithDefault(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.LastExecError(), ErrorExecErrNotFound)
	assert.Equal(t, result.FirstExecError(), ErrorExecErrNotFound)
	assert.Equal(t, result.ExecErrors(), []error{})
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetry_TryOnConflictSuccess(t *testing.T) {
	r := New(nil)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return "lee", nil
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(1))
}
func TestRetry_TryOnConflictContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := NewConfig().WithContext(ctx)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), context.Canceled)
	assert.Equal(t, result.Count(), int64(0))
}

func TestRetry_TryOnConflictCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := NewConfig().WithContext(ctx)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return "lee", nil
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), context.Canceled)
	assert.Equal(t, result.Count(), int64(0))
}

func TestRetry_TryOnConflictCallback(t *testing.T) {
	cfg := NewConfig().WithDetail(true).WithAttempts(5).WithCallback(&callback{})
	e := errors.New("test")

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), false)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e, e, e, e, e})
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(5))
}

func TestRetry_TryOnConflictRetryIf(t *testing.T) {
	e := errors.New("test")

	retryIf := func(err error) bool {
		return !errors.Is(err, e)
	}

	cfg := NewConfig().WithRetryIfFunc(retryIf)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryIf)
	assert.Equal(t, result.Count(), int64(1))
}

func TestRetry_TryOnConflictRetryIfExceeded(t *testing.T) {
	cfg := NewConfig().WithAttempts(2)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetry_TryOnConflictAttemptsByError(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	cfg := NewConfig().WithAttemptsByError(m)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsByErrorExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetry_TryOnConflictAttemptsExceeded(t *testing.T) {
	cfg := NewConfig().WithAttempts(2)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflictVal(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetry_TryOnConflictMultiRetryableFuncs(t *testing.T) {
	cfg := NewConfig().WithCallback(&callback{})

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc1 := func() (any, error) {
		return nil, errors.New("testFunc1")
	}

	testFunc2 := func() (any, error) {
		return nil, errors.New("testFunc2")
	}

	result := r.TryOnConflictVal(testFunc1)
	assert.NotNil(t, result)
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(defaultAttempts))

	result = r.TryOnConflictVal(testFunc2)
	assert.NotNil(t, result)
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(defaultAttempts))
}

func TestRetry_TryOnConflictMultiRetryableFuncsParallel(t *testing.T) {
	cfg := NewConfig().WithCallback(&callback{})

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc1 := func() (any, error) {
		return nil, errors.New("testFunc1")
	}

	testFunc2 := func() (any, error) {
		return nil, errors.New("testFunc2")
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		result1 := r.TryOnConflictVal(testFunc1)
		assert.NotNil(t, result1)
		assert.Equal(t, result1.TryError(), ErrorRetryAttemptsExceeded)
		assert.Equal(t, result1.Count(), int64(defaultAttempts))
	}()

	go func() {
		defer wg.Done()
		result2 := r.TryOnConflictVal(testFunc2)
		assert.NotNil(t, result2)
		assert.Equal(t, result2.TryError(), ErrorRetryAttemptsExceeded)
		assert.Equal(t, result2.Count(), int64(defaultAttempts))
	}()

	wg.Wait()
}

func TestRetry_ZeroConfig(t *testing.T) {
	cfg := NewConfig()
	r := New(cfg)
	assert.NotNil(t, r)

	result := r.TryOnConflictVal(func() (any, error) {
		return nil, nil
	})

	assert.True(t, result.IsSuccess())
	assert.Equal(t, int64(1), result.Count())
}

func TestRetry_MultipleErrorTypes(t *testing.T) {
	errType1 := errors.New("type1")
	errType2 := errors.New("type2")

	m := map[error]uint64{
		errType1: 2,
		errType2: 3,
	}

	cfg := NewConfig().WithAttemptsByError(m).WithDetail(true)
	r := New(cfg)

	count := 0
	result := r.TryOnConflictVal(func() (any, error) {
		count++
		if count <= 3 {
			return nil, errType1
		}
		return nil, errType2
	})

	assert.Equal(t, int64(3), result.Count())
	assert.Equal(t, ErrorRetryAttemptsByErrorExceeded, result.TryError())

	errors := result.ExecErrors()
	assert.NotNil(t, errors)
	assert.NotEmpty(t, errors)

	assert.Equal(t, 3, len(errors))
	for _, err := range errors {
		assert.Equal(t, errType1, err)
	}
}

func TestRetry_ConcurrentStress(t *testing.T) {
	cfg := NewConfig().WithAttempts(5)
	r := New(cfg)

	var wg sync.WaitGroup
	concurrent := 10
	wg.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func(id int) {
			defer wg.Done()
			result := r.TryOnConflictVal(func() (any, error) {
				if id%2 == 0 {
					return fmt.Sprintf("success-%d", id), nil
				}
				return nil, fmt.Errorf("error-%d", id)
			})

			if id%2 == 0 {
				assert.True(t, result.IsSuccess())
				assert.Equal(t, fmt.Sprintf("success-%d", id), result.Data())
			} else {
				assert.False(t, result.IsSuccess())
			}
		}(i)
	}

	wg.Wait()
}

func TestRetry_ConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected error
	}{
		{
			name:     "zero attempts",
			cfg:      NewConfig().WithAttempts(0),
			expected: ErrorRetryAttemptsExceeded,
		},
		{
			name:     "max attempts",
			cfg:      NewConfig().WithAttempts(math.MaxUint64),
			expected: ErrorRetryAttemptsExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.cfg)
			result := r.TryOnConflictVal(func() (any, error) {
				return nil, errors.New("test")
			})
			assert.Equal(t, tt.expected, result.TryError())
		})
	}
}
