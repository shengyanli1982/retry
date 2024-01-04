package retry

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type callback struct{}

func (cb *callback) OnRetry(count int64, delay time.Duration, err error) {
	fmt.Println("OnRetry", count, delay.String(), err)
}

func TestNewRetry_Standard(t *testing.T) {
	r := newRetry(nil)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return "lee", nil
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.LastExecError(), ErrorExecErrNotFound)
	assert.Equal(t, result.FirstExecError(), ErrorExecErrNotFound)
	assert.Equal(t, result.ExecErrors(), []error{})
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(1))
}

func TestNewRetry_TotalCountExceeded(t *testing.T) {
	cfg := NewConfig().WithDetail(true).WithAttempts(2)
	e := errors.New("test")

	r := newRetry(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), false)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e, e})
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestNewRetry_SpecError(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	cfg := NewConfig().WithAttemptsByError(m).WithDetail(true)

	r := newRetry(cfg)
	assert.NotNil(t, r)

	count := 0
	testFunc := func() (any, error) {
		if count > 0 {
			return "lee", nil
		} else {
			count++
			return nil, e
		}
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), true)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e})
	assert.Equal(t, result.Data(), "lee")
	assert.Equal(t, result.Count(), int64(2))
}

func TestNewRetry_SpecErrorCountExceeded(t *testing.T) {
	m := map[error]uint64{}
	e := errors.New("test")
	m[e] = 1

	cfg := NewConfig().WithAttemptsByError(m).WithDetail(true)

	r := newRetry(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), false)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e, e})
	assert.Equal(t, result.TryError(), ErrorRetryAttemptsByErrorExceeded)
	assert.Equal(t, result.Count(), int64(2))
}

func TestNewRetry_RetryIf(t *testing.T) {
	e := errors.New("test")

	retryIf := func(err error) bool {
		return !errors.Is(err, e)
	}

	cfg := NewConfig().WithDetail(true).WithAttempts(2).WithRetryIf(retryIf)

	r := newRetry(cfg)
	assert.NotNil(t, r)

	testFunc := func() (any, error) {
		return nil, e
	}

	result := r.TryOnConflict(testFunc)
	assert.NotNil(t, result)

	assert.Equal(t, result.IsSuccess(), false)
	assert.Equal(t, result.LastExecError(), e)
	assert.Equal(t, result.FirstExecError(), e)
	assert.Equal(t, result.ExecErrors(), []error{e})
	assert.Equal(t, result.TryError(), ErrorRetryIf)
	assert.Equal(t, result.Count(), int64(1))
}

func TestNewRetry_Callback(t *testing.T) {
	cfg := NewConfig().WithDetail(true).WithAttempts(5).WithCallback(&callback{})
	e := errors.New("test")

	r := newRetry(cfg)
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
