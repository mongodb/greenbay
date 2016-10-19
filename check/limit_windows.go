// +build windows

package check

import (
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"golang.org/x/sys/windows/registry"
)

// because the limit_*.go files are only built on some platforms, we
// define a "limitValueCheckTable" function in all files which returns
// a map of "limit name" to check function.

func limitValueCheckTable() map[string]limitValueCheck {
	return map[string]limitValueCheck{
		"irp-stack-size": irpStackSize,
	}
}

func irpStackSize(value int) (bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\services\LanmanServer\Parameters`, registry.QUERY_VALUE)
	if err != nil {
		return false, errors.Wrap(err, "problem opening registry key")
	}
	defer grip.Warning(key.Close())

	irpStackSize, _, err := key.GetIntegerValue("IRPStackSize")
	if err != nil {
		return false, errors.Wrap(err, "problem getting value of IRPStackSize Value")
	}

	if irpStackSize != value {
		return false, errors.Errorf("IRPStackSize should be %d but is %d", value, irpStackSize)
	}

	return nil

}
