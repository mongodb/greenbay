// +build linux freebsd solaris darwin

package check

import (
	"syscall"

	"github.com/pkg/errors"
)

// because the limit_*.go files are only built on some platforms, we
// define a "limitValueCheckTable" function in all files which returns
// a map of "limit name" to check function.

func limitValueCheckTable() map[string]limitValueCheck {
	return map[string]limitValueCheck{
		"open-files":   limitCheckFactory("open-files", syscall.RLIMIT_NOFILE),
		"address-size": limitCheckFactory("address-size", syscall.RLIMIT_AS),
	}
}

func limitCheckFactory(name string, resource int) limitValueCheck {
	return func(value int) (bool, error) {
		limits := &syscall.Rlimit{}
		err := syscall.Getrlimit(resource, limits)
		if err != nil {
			return false, errors.Wrapf(err, "problem finding %s limit", name)
		}

		if limits.Max < uint64(value) {
			return false, errors.Errorf("%s limit is %d which is less than %d",
				name, limits.Max, value)
		}

		return true, nil
	}
}
