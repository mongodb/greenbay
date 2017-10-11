// +build linux

package operations

import "github.com/mongodb/grip/send"

func setupSystemdLogging() (send.Sender, error) {
	return send.MakeSystemdLogger()
}
