// +build linux

package operations

import "github.com/tychoish/grip/send"

func setupSystemdLogging() send.Sender {
	return send.MakeSystemdLogger()
}
