package retry

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackOffs(t *testing.T) {
	t0 := 3
	fixed := FixBackOff(int64(t0))
	assert.Equal(t, fixed, time.Duration(t0)*baseTimeDuration)
	exponential := ExponentialBackOff(int64(t0))
	assert.Equal(t, exponential, time.Duration(math.Exp2(float64(t0)))*baseTimeDuration)
	combined := CombineBackOffs(FixBackOff, ExponentialBackOff)
	assert.Equal(t, combined(int64(t0)), time.Duration(t0+int(math.Exp2(float64(t0))))*baseTimeDuration)
}
