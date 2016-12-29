// +build windows

package operations

import (
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/send"
)

func setupSyslogLogging() send.Sender {
	grip.Warning("syslog is not supported on this platform, falling back to stdout logging.")
	return send.NewNative()
}
