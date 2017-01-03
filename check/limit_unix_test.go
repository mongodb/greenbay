// +build linux freebsd solaris darwin

package check

import (
	"fmt"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitCheckFactoryErrorConditions(t *testing.T) {
	assert := assert.New(t)
	checks := []limitValueCheck{
		limitCheckFactory("a", -1000000000000000000, 10),
		limitCheckFactory("d", -1, 10),
	}

	if runtime.GOOS == "darwin" {
		checks = append(checks,
			limitCheckFactory("b", syscall.RLIMIT_CPU, 10),
			limitCheckFactory("c", 0, 10))
	}

	for idx, c := range checks {
		result, err := c(-idx)
		assert.Error(err, fmt.Sprintf("%+v: %d", err, idx))
		assert.False(result)
	}
}
