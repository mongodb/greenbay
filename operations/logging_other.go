// +build !linux

package operations

import (
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
)

func setupSystemdLogging() send.Sender {
	grip.Warning("systemd logging is not supported on this platform, falling back to stdout logging.")
	return send.MakeNative()
}
