package check

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func scriptCompilerInterfaceFactoryTable() map[string]compilerFactory {
	factory := func(path string) compilerFactory {
		return func() compiler {
			return &compileScript{
				bin: path,
			}
		}
	}

	return map[string]compilerFactory{
		"run-program-system-python":           factory("/usr/bin/python"),
		"run-program-system-python2":          factory("/usr/bin/python2"),
		"run-program-system-python3":          factory("/usr/bin/python3"),
		"run-program-system-pypy":             factory("/usr/bin/pypy"),
		"run-program-system-usr-local-python": factory("/usr/local/bin/python"),
		"run-bash-script":                     factory("/bin/bash"),
		"run-sh-script":                       factory("/bin/sh"),
		"run-dash-script":                     factory("/bin/dash"),
		"run-zsh-script":                      factory("/bin/dash"),
	}
}

type compileScript struct {
	bin string
}

func (c *compileScript) Validate() error {
	if c.bin == "" {
		return errors.New("no script interpreter")
	}

	if _, err := os.Stat(c.bin); os.IsNotExist(err) {
		return errors.Errorf("script interpreter '%s' does not exist", c.bin)
	}

	return nil
}

func (c *compileScript) Compile(testBody string, _ ...string) error {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return errors.Wrap(err, "problem writing test")
	}

	defer os.Remove(sourceName)

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running script script with command: %s", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "problem build/running test script %s: %s", sourceName,
			string(output))
	}

	return nil
}

func (c *compileScript) CompileAndRun(testBody string, _ ...string) (string, error) {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return "", errors.Wrap(err, "problem writing test")
	}

	defer os.Remove(sourceName)

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running script script with command: %s", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return output, errors.Wrapf(err, "problem running test script %s", sourceName)
	}

	return strings.Trim(output, "\n \t"), nil
}
