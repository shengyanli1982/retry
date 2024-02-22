package retry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type callback struct{}

func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

func TestRetryDo(t *testing.T) {
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

func TestRetryDoWithDefault(t *testing.T) {
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

func TestRetryTryOnConflictSuccess(t *testing.T) {
	r := New(nil)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return "lee", nil
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(1))
}
func TestRetryTryOnConflictContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := NewConfig().WithContext(ctx)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), context.Canceled)
	assert.Equal(t, result.Count(), int64(0))
}

func TestRetryTryOnConflictCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := NewConfig().WithContext(ctx)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return "lee", nil
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), context.Canceled)
	assert.Equal(t, result.Count(), int64(0))
}

func TestRetryTryOnConflictCallback(t *testing.T) {
	cfg := NewConfig().WithDetail(true).WithAttempts(5).WithCallback(&callback{})
	e := errors.New("test")

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), false)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e, e, e, e, e})
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(5))
}

func TestRetryTryOnConflictRetryIf(t *testing.T) {
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

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryIf)
	assert.Equal(t, result.Count(), int64(1))
}

func TestRetryTryOnConflictRetryIfExceeded(t *testing.T) {
	cfg := NewConfig().WithAttempts(2)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetryTryOnConflictAttemptsByError(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	cfg := NewConfig().WithAttemptsByError(m)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsByErrorExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetryTryOnConflictAttemptsExceeded(t *testing.T) {
	cfg := NewConfig().WithAttempts(2)

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, errors.New("test")
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestRetryTryOnConflictMultiRetryableFuncs(t *testing.T) {
	cfg := NewConfig().WithCallback(&callback{})

	r := New(cfg)
	assert.NotNil(t, r)

	testFunc1 := func() (any, error) {
		return nil, errors.New("testFunc1")
	}

	testFunc2 := func() (any, error) {
		return nil, errors.New("testFunc2")
	}

	result := r.TryOnConflict(testFunc1)
	assert.NotNil(t, result)
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(defaultAttempts))

	result = r.TryOnConflict(testFunc2)
	assert.NotNil(t, result)
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(defaultAttempts))
}

func TestRetryTryOnConflictMultiRetryableFuncsParallel(t *testing.T) {
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
		result1 := r.TryOnConflict(testFunc1)
		assert.NotNil(t, result1)
		assert.Equal(t, result1.TryError(), ErrorRetryAttemptsExceeded)
		assert.Equal(t, result1.Count(), int64(defaultAttempts))
	}()

	go func() {
		defer wg.Done()
		result2 := r.TryOnConflict(testFunc2)
		assert.NotNil(t, result2)
		assert.Equal(t, result2.TryError(), ErrorRetryAttemptsExceeded)
		assert.Equal(t, result2.Count(), int64(defaultAttempts))
	}()

	wg.Wait()
}
