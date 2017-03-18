// +build linux

package operations

import "github.com/mongodb/grip/send"

func setupSystemdLogging() send.Sender {
	return send.MakeSystemdLogger()
}
