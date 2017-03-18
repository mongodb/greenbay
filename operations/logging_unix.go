// +build linux freebsd solaris darwin

package operations

import "github.com/mongodb/grip/send"

func setupSyslogLogging() send.Sender {
	return send.MakeLocalSyslogLogger()
}
