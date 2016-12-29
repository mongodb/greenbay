// +build linux freebsd solaris darwin

package operations

import "github.com/tychoish/grip/send"

func setupSyslogLogging() send.Sender {
	return send.MakeLocalSyslogLogger()
}
