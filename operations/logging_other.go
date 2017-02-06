// +build !linux

package operations

import (
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/send"
)

func setupSystemdLogging() send.Sender {
	grip.Warning("systemd logging is not supported on this platform, falling back to stdout logging.")
	return send.MakeNative()
}
