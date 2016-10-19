// +build linux freebsd solaris darwin

package check

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func compilerInterfaceFactoryTable() map[string]compilerFactory {
	factory := func(path string) func() compiler {
		return func() compiler {
			return compileGCC{
				bin: path,
			}
		}
	}

	return map[string]compilerFactory{
		"compile-gcc-auto":     gccCompilerAuto,
		"compile-gcc-system":   factory("gcc"),
		"compile-toolchain-v2": factory("/opt/mongodbtoolchain/v2/bin/gcc"),
		"compile-toolchain-v1": factory("/opt/mongodbtoolchain/v1/bin/gcc"),
		"compile-toolchain-v0": factory("/opt/mongodbtoolchain/bin/gcc"),
	}
}

type compileGCC struct {
	bin string
}

func gccCompilerAuto() compiler {
	c := compileGCC{}

	paths := []string{
		"/opt/mongodbtoolchain/v2/bin/gcc",
		"/opt/mongodbtoolchain/v1/bin/gcc",
		"/opt/mongodbtoolchain/bin/gcc",
		"/usr/bin/gcc",
		"/usr/local/bin/gcc",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			c.bin = path
			break
		}
	}

	if c.bin == "" {
		c.bin = "gcc"
	}

	return c
}

func (c compileGCC) Validate() error {
	if c.bin == "" {
		return errors.New("no compiler specified")
	}

	if _, err := os.Stat(c.bin); os.IsNotExist(err) {
		return errors.Errorf("compiler binary '%s' does not exist", c.bin)
	}

	return nil
}

func (c compileGCC) Compile(testBody string, cFlags ...string) error {
	outputName, sourceName, err := writeTestBody(testBody, "c")
	outputName += ".o"
	defer grip.CatchWarning(os.Remove(outputName))

	cmd := exec.Command(c.bin, "-Werror", "-o", outputName, "-c", sourceName)
	grip.Infof("running build command: %s", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "problem compiling test body: %s", string(output))
	}

	return nil
}

func (c compileGCC) CompileAndRun(testBody string, cFlags ...string) (string, error) {
	outputName, sourceName, err := writeTestBody(testBody, "c")
	defer grip.CatchWarning(os.Remove(outputName))

	argv := []string{"-Werror", "-o", outputName}
	argv = append(argv, sourceName)
	argv = append(argv, cFlags...)

	cmd := exec.Command(c.bin, argv...)
	grip.Infof("running build command: %s", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), errors.Wrap(err, "problem compiling test")
	}

	cmd = exec.Command(outputName)
	grip.Infof("running test command: %s", strings.Join(cmd.Args, " "))
	out, err = cmd.CombinedOutput()
	if err != nil {
		return string(out), errors.Wrap(err, "problem running test program")
	}

	return strings.Trim(string(out), "\t\n "), nil
}
