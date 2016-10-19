package check

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func goCompilerIterfaceFactoryTable() map[string]compilerFactory {
	factory := func(path string) compilerFactory {
		return func() compiler {
			return compileGolang{
				bin: path,
			}
		}
	}

	return map[string]compilerFactory{
		"compile-go-auto":            goCompilerAuto,
		"compile-opt-go-default":     factory("/opt/go/bin/go"),
		"compile-toolchain-gccgo-v2": factory("/opt/mongodbtoolchain/v2/bin/go"),
		"compile-usr-local-go":       factory("/usr/local/go"),
		"compile-user-local-go":      factory("/usr/bin/go"),
	}
}

func goCompilerAuto() compiler {
	paths := []string{
		"/opt/go/bin/go",
		"/opt/mongodbtoolchain/v2/bin/go",
		"/usr/bin/go",
		"/usr/local/bin/go",
	}
	c := compileGolang{}

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			c.bin = path
			break
		}
	}

	if c.bin == "" {
		c.bin = "go"
	}

	return c
}

type compileGolang struct {
	bin string
}

func (c compileGolang) Validate() error {
	if c.bin == "" {
		return errors.New("no go binary specified")
	}

	if _, err := os.Stat(c.bin); os.IsNotExist(err) {
		return errors.Errorf("go binary '%s' does not exist", c.bin)
	}

	return nil
}

func (c compileGolang) Compile(testBody string, _ ...string) error {
	_, source, err := writeTestBody(testBody, "go")
	if err != nil {
		return errors.Wrap(err, "problem writing test to temporary file")
	}

	cmd := exec.Command(c.bin, "build", source)
	grip.Infof("running build command: %s", cmd.Args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "problem compiling go test: %s", string(out))
	}

	return nil
}

func (c compileGolang) CompileAndRun(testBody string, _ ...string) (string, error) {
	_, source, err := writeTestBody(testBody, "go")
	if err != nil {
		return "", errors.Wrap(err, "problem writing test to temporary file")
	}

	cmd := exec.Command(c.bin, "run", source)
	grip.Infof("running build command: %s", cmd.Args)

	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return output, errors.Wrapf(err, "problem running go program: %s", output)
	}

	output = strings.Trim(output, "\t\n ")
	return output, nil
}
