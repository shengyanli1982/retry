package retry

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackOffs(t *testing.T) {
	t0 := 3
	fix := FixBackOff(int64(t0))
	assert.Equal(t, fix, time.Duration(t0)*baseTimeDuration)
	exp := ExponentialBackOff(int64(t0))
	assert.Equal(t, exp, time.Duration(math.Exp2(float64(t0)))*baseTimeDuration)
	combine := CombineBackOffs(FixBackOff, ExponentialBackOff)
	assert.Equal(t, combine(int64(t0)), time.Duration(t0+int(math.Exp2(float64(t0))))*baseTimeDuration)
}
