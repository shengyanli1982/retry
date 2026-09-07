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
	e := errors.New("test")
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
	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsExceeded))
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

	assert.True(t, errors.Is(result.TryError(), ErrorRetryIf))
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

	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsExceeded))
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

	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsByErrorExceeded))
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

	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsExceeded))
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
	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsExceeded))
	assert.Equal(t, result.Count(), int64(defaultAttempts))

	result = r.TryOnConflictVal(testFunc2)
	assert.NotNil(t, result)
	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsExceeded))
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
		assert.True(t, errors.Is(result1.TryError(), ErrorRetryAttemptsExceeded))
		assert.Equal(t, result1.Count(), int64(defaultAttempts))
	}()

	go func() {
		defer wg.Done()
		result2 := r.TryOnConflictVal(testFunc2)
		assert.NotNil(t, result2)
		assert.True(t, errors.Is(result2.TryError(), ErrorRetryAttemptsExceeded))
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
	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsByErrorExceeded))

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

	for i := range concurrent {
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
	// Q7 定案行为：巨大 attempts（>=65535）被钳制为 65534 而非重置为 3，
	// 穷尽 65534 次（指数退避单次即达百年级）在测试时间轴内不可达，
	// 故用短超时 context 约束执行：首次失败后退避（>=700ms）远超超时（100ms），ctx 先终止重试。
	// Settled Q7 behavior: huge attempts (>=65535) are clamped to 65534 instead of reset to 3;
	// exhausting 65534 tries (a single exponential backoff step already spans centuries) is unreachable in test time,
	// so a short-timeout context bounds the run: after the first failure the backoff (>=700ms) far exceeds the timeout (100ms) and ctx ends the retry first.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tests := []struct {
		name     string
		cfg      *Config
		expected error
	}{
		{
			name:     "max attempts",
			cfg:      NewConfig().WithAttempts(math.MaxUint64).WithContext(timeoutCtx),
			expected: context.DeadlineExceeded,
		},
		{
			name:     "zero attempts",
			cfg:      NewConfig().WithAttempts(0),
			expected: ErrorRetryAttemptsExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.cfg)
			result := r.TryOnConflictVal(func() (any, error) {
				return nil, errors.New("test")
			})
			assert.True(t, errors.Is(result.TryError(), tt.expected))
		})
	}
}

// TestNilFuncInterfaceExitsReturnUntypedNil 验证 fn==nil 时三个接口出口返回 untyped nil，
// 使惯用的 res == nil 判空生效（不得把 nil *Result 装箱进 RetryResult 接口）。
// 注意：必须用 == nil 直接比较；testify 的 assert.Nil 走反射判空，对 typed-nil 同样为真，无鉴别力。
// TestNilFuncInterfaceExitsReturnUntypedNil verifies the three interface exits return untyped nil when fn==nil,
// so the idiomatic res == nil check works (a nil *Result must not be boxed into the RetryResult interface).
// Note: must use == nil directly; testify's assert.Nil uses reflection and also reports typed-nil as nil.
func TestNilFuncInterfaceExitsReturnUntypedNil(t *testing.T) {
	assert.True(t, Do(nil, nil) == nil, "Do(nil, nil) 必须返回 untyped nil")
	assert.True(t, DoWithDefault(nil) == nil, "DoWithDefault(nil) 必须返回 untyped nil")
	assert.True(t, New(nil).TryOnConflictVal(nil) == nil, "TryOnConflictVal(nil) 必须返回 untyped nil")
}

// TestNewDoesNotMutateCallerConfig 验证 New 对传入 Config 做副本归一化，永不写调用方对象：
// 含无效字段的 cfg 经 New 后各字段必须保持原值（副作用不外泄）。
// TestNewDoesNotMutateCallerConfig verifies New normalizes a copy of the passed Config and never writes the caller's object:
// a cfg with invalid fields must keep all field values unchanged after New (no side-effect leakage).
func TestNewDoesNotMutateCallerConfig(t *testing.T) {
	cfg := &Config{} // 零值 Config：ctx/callback/attempts/map/delay/retryIfFunc/backoffFunc 均属待归一化字段
	_ = New(cfg)

	assert.Nil(t, cfg.ctx, "New 不得写入调用方 cfg.ctx")
	assert.Nil(t, cfg.callback, "New 不得写入调用方 cfg.callback")
	assert.Equal(t, uint64(0), cfg.attempts, "New 不得写入调用方 cfg.attempts")
	assert.Nil(t, cfg.attemptsByError, "New 不得写入调用方 cfg.attemptsByError")
	assert.Equal(t, time.Duration(0), cfg.delay, "New 不得写入调用方 cfg.delay")
	assert.Nil(t, cfg.retryIfFunc, "New 不得写入调用方 cfg.retryIfFunc")
	assert.Nil(t, cfg.backoffFunc, "New 不得写入调用方 cfg.backoffFunc")
}

// TestConcurrentNewOnSharedConfig 验证多 goroutine 对同一零值 *Config 并发 New+Do 无数据竞争
// （配合 go test -race 运行；修复前 isConfigValid 并发写同一批字段会被 race detector 捕获）。
// TestConcurrentNewOnSharedConfig verifies concurrent New+Do on one shared zero-value *Config is race-free
// (run with go test -race; before the fix, isConfigValid wrote the same fields concurrently and the race detector caught it).
func TestConcurrentNewOnSharedConfig(t *testing.T) {
	shared := &Config{}

	var wg sync.WaitGroup
	wg.Add(8)
	for range 8 {
		go func() {
			defer wg.Done()
			result := Do(func() (any, error) { return "ok", nil }, shared)
			assert.NotNil(t, result)
			assert.True(t, result.IsSuccess())
		}()
	}
	wg.Wait()

	// 并发场景下调用方对象同样不得被写入
	assert.Equal(t, uint64(0), shared.attempts)
	assert.Nil(t, shared.ctx)
}

// TestWithAttemptsByErrorCopiesMap 验证 WithAttemptsByError 在入口拷贝用户 map，库持有私有副本：
// With 之后修改原 map 不得影响重试预算。
// TestWithAttemptsByErrorCopiesMap verifies WithAttemptsByError copies the user map at the entry so the library holds a private copy:
// modifying the original map after With must not affect the retry budget.
func TestWithAttemptsByErrorCopiesMap(t *testing.T) {
	e := errors.New("test")
	m := map[error]uint64{e: 1}

	cfg := NewConfig().WithAttemptsByError(m)
	m[e] = 100 // With 之后修改原 map：若库别名该 map，预算将被放大到 100

	r := New(cfg)
	result := r.TryOnConflictVal(func() (any, error) {
		return nil, e
	})

	// 预算应仍为快照时的 1：第 2 次执行即触发 ByErrorExceeded 中止；
	// 若别名了修改后的原 map，则会在 attempts=3 处以 AttemptsExceeded 中止（count=3）
	assert.True(t, errors.Is(result.TryError(), ErrorRetryAttemptsByErrorExceeded), "预算必须来自入口拷贝时的快照")
	assert.Equal(t, int64(2), result.Count())
}

// TestAttemptsExceededPreservesRootCause 验证 attempts 耗尽路径与 per-error 中止路径
// 均以双 %w 保留 fn 的原始错误根因：errors.Is 须同时命中哨兵与原始错误（与 retryIf 路径语义一致）。
// TestAttemptsExceededPreservesRootCause verifies the attempts-exhausted path and the per-error abort path
// both preserve fn's original error as root cause via double %w: errors.Is must hit both the sentinel and the original error (consistent with the retryIf path).
func TestAttemptsExceededPreservesRootCause(t *testing.T) {
	e := errors.New("root cause")

	t.Run("attempts exhausted", func(t *testing.T) {
		cfg := NewConfig().WithAttempts(2)
		result := New(cfg).TryOnConflictVal(func() (any, error) {
			return nil, e
		})
		tryError := result.TryError()
		assert.True(t, errors.Is(tryError, ErrorRetryAttemptsExceeded), "errors.Is 必须命中哨兵 ErrorRetryAttemptsExceeded")
		assert.True(t, errors.Is(tryError, e), "errors.Is 必须命中 fn 的原始错误（根因保留）")
	})

	t.Run("per-error budget exhausted", func(t *testing.T) {
		cfg := NewConfig().WithAttemptsByError(map[error]uint64{e: 1})
		result := New(cfg).TryOnConflictVal(func() (any, error) {
			return nil, e
		})
		tryError := result.TryError()
		assert.True(t, errors.Is(tryError, ErrorRetryAttemptsByErrorExceeded), "errors.Is 必须命中哨兵 ErrorRetryAttemptsByErrorExceeded")
		assert.True(t, errors.Is(tryError, e), "errors.Is 必须命中 fn 的原始错误（根因保留）")
	})
}

// countingCallback 记录每次 OnRetry 回调收到的 count 值
// countingCallback records the count value received by each OnRetry callback
type countingCallback struct {
	mu     sync.Mutex
	counts []int64
}

func (cb *countingCallback) OnRetry(count int64, delay time.Duration, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.counts = append(cb.counts, count)
}

// TestOnRetryCalledOnlyForActualRetries 验证 OnRetry 只在确定要发起下一次重试时回调：
// attempts=3 全失败 → 实际只发生 2 次重试，OnRetry 恰好回调 2 次（count=1,2），
// 中止前的最后一次失败不得回调；per-error 预算中止路径同理。
// TestOnRetryCalledOnlyForActualRetries verifies OnRetry is called only when a next retry will actually happen:
// attempts=3 all failing → only 2 actual retries occur, so OnRetry must be called exactly 2 times (count=1,2);
// the final failing attempt before abort must not trigger it; the per-error budget abort path behaves the same.
func TestOnRetryCalledOnlyForActualRetries(t *testing.T) {
	e := errors.New("test")
	alwaysFail := func() (any, error) { return nil, e }

	t.Run("attempts exhausted", func(t *testing.T) {
		cb := &countingCallback{}
		cfg := NewConfig().WithAttempts(3).WithCallback(cb)
		result := New(cfg).TryOnConflictVal(alwaysFail)
		assert.Equal(t, int64(3), result.Count())
		assert.Equal(t, []int64{1, 2}, cb.counts, "OnRetry 只应为实际发生的 2 次重试回调")
	})

	t.Run("per-error budget abort", func(t *testing.T) {
		cb := &countingCallback{}
		cfg := NewConfig().WithAttemptsByError(map[error]uint64{e: 1}).WithCallback(cb)
		result := New(cfg).TryOnConflictVal(alwaysFail)
		assert.Equal(t, int64(2), result.Count())
		assert.Equal(t, []int64{1}, cb.counts, "per-error 预算耗尽的中止不得回调 OnRetry")
	})
}

// TestAttemptsClamping 验证 attempts 归一化语义（Q7 定案）：
// 0 → 重置为默认 3；>=65535 → 钳制为 65534（math.MaxUint16-1）而非重置为 3；合法值原样保留。
// TestAttemptsClamping verifies attempts normalization semantics (settled Q7):
// 0 → reset to default 3; >=65535 → clamped to 65534 (math.MaxUint16-1) instead of reset to 3; valid values kept as-is.
func TestAttemptsClamping(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{"zero falls back to default", 0, defaultAttempts},
		{"one is kept", 1, 1},
		{"max valid is kept", math.MaxUint16 - 1, math.MaxUint16 - 1},
		{"boundary clamps", math.MaxUint16, math.MaxUint16 - 1},
		{"large value clamps", 100000, math.MaxUint16 - 1},
		{"max uint64 clamps", math.MaxUint64, math.MaxUint16 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(NewConfig().WithAttempts(tt.input))
			assert.Equal(t, tt.expected, r.config.attempts)
		})
	}
}

func BenchmarkNew(b *testing.B) {
	cfg := NewConfig()
	b.ResetTimer()
	for range b.N {
		_ = New(cfg)
	}
}

func BenchmarkNew_NilConfig(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = New(nil)
	}
}

func BenchmarkTryOnConflict_Success(b *testing.B) {
	cfg := NewConfig().WithInitDelay(0).WithJitter(0).WithFactor(0)
	r := New(cfg)
	fn := func() (any, error) { return 42, nil }
	b.ResetTimer()
	for range b.N {
		_ = r.TryOnConflict(fn)
	}
}

func BenchmarkTryOnConflict_Retry3(b *testing.B) {
	cfg := NewConfig().WithAttempts(5).WithInitDelay(time.Nanosecond).WithJitter(0).WithFactor(0).
		WithBackOffFunc(func(_ int64) time.Duration { return 0 })
	r := New(cfg)
	b.ResetTimer()
	for range b.N {
		count := 0
		_ = r.TryOnConflict(func() (any, error) {
			count++
			if count >= 3 {
				return "ok", nil
			}
			return nil, errors.New("retry")
		})
	}
}

func BenchmarkTryOnConflict_AllFail(b *testing.B) {
	cfg := NewConfig().WithAttempts(3).WithInitDelay(time.Nanosecond).WithJitter(0).WithFactor(0).
		WithBackOffFunc(func(_ int64) time.Duration { return 0 })
	r := New(cfg)
	fn := func() (any, error) { return nil, errors.New("fail") }
	b.ResetTimer()
	for range b.N {
		_ = r.TryOnConflict(fn)
	}
}

func BenchmarkTryOnConflict_WithDetail(b *testing.B) {
	cfg := NewConfig().WithAttempts(3).WithInitDelay(time.Nanosecond).WithJitter(0).WithFactor(0).
		WithBackOffFunc(func(_ int64) time.Duration { return 0 }).WithDetail(true)
	r := New(cfg)
	fn := func() (any, error) { return nil, errors.New("fail") }
	b.ResetTimer()
	for range b.N {
		_ = r.TryOnConflict(fn)
	}
}

func BenchmarkDo(b *testing.B) {
	cfg := NewConfig().WithInitDelay(0).WithJitter(0).WithFactor(0)
	fn := func() (any, error) { return 42, nil }
	b.ResetTimer()
	for range b.N {
		_ = Do(fn, cfg)
	}
}

func BenchmarkDoWithDefault(b *testing.B) {
	fn := func() (any, error) { return 42, nil }
	b.ResetTimer()
	for range b.N {
		_ = DoWithDefault(fn)
	}
}

func BenchmarkTryOnConflict_WithAttemptsByError(b *testing.B) {
	e := errors.New("specific")
	cfg := NewConfig().WithAttempts(10).WithAttemptsByError(map[error]uint64{e: 3}).
		WithInitDelay(time.Nanosecond).WithJitter(0).WithFactor(0).
		WithBackOffFunc(func(_ int64) time.Duration { return 0 })
	r := New(cfg)
	fn := func() (any, error) { return nil, e }
	b.ResetTimer()
	for range b.N {
		_ = r.TryOnConflict(fn)
	}
}
