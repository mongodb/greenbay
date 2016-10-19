package check

import (
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

type compiler interface {
	Validate() error
	Compile(string, ...string) error
	CompileAndRun(string, ...string) (string, error)
}

type compilerFactory func() compiler

func writeTestBody(testBody, ext string) (string, string, error) {
	testFile, err := ioutil.TempFile(os.TempDir(), "testBody_")
	if err != nil {
		return "", "", err
	}

	baseName := testFile.Name()
	sourceName := strings.Join([]string{baseName, ext}, ".")

	if runtime.GOOS == "windows" {
		testBody = strings.Replace(testBody, "\n", "\r\n", -1)
	}

	_, err = testFile.Write([]byte(testBody))
	if err != nil {
		return "", "", errors.Wrap(err, "problem writing test to file")
	}
	defer grip.CatchWarning(testFile.Close())

	if err = os.Rename(baseName, sourceName); err != nil {
		return "", "", errors.Wrap(err, "problem renaming file")

	}

	return baseName, sourceName, nil
}

func registerCompileChecks() {
	compileCheckFactoryFactory := func(name string, c compiler, shouldRun bool) func() amboy.Job {
		return func() amboy.Job {
			return &compileCheck{
				Base:          NewBase(name, 0),
				shouldRunCode: shouldRun,
				compiler:      c,
			}
		}
	}

	registrar := func(table map[string]compilerFactory) {
		var jobName string
		for name, factory := range table {
			for _, shouldRun := range []bool{true, false} {
				if shouldRun {
					jobName = strings.Replace(name, "compile-", "compile-and-run-", 1)
				} else {
					jobName = name
				}

				registry.AddJobType(jobName,
					compileCheckFactoryFactory(jobName, factory(), shouldRun))
			}
		}
	}

	registrar(compilerInterfaceFactoryTable())
	registrar(goCompilerIterfaceFactoryTable())
}

type compileCheck struct {
	Source        string `bson:"source" json:"source" yaml:"source"`
	*Base         `bson:"metadata" json:"metadata" yaml:"metadata"`
	shouldRunCode bool
	compiler      compiler
}

func (c *compileCheck) Run() {
	c.startTask()
	defer c.markComplete()

	c.setState(true)

	if err := c.compiler.Validate(); err != nil {
		c.setState(false)
		c.addError(err)
		return
	}

	if c.shouldRunCode {
		if output, err := c.compiler.CompileAndRun(c.Source); err != nil {
			c.setState(false)
			c.addError(err)
			c.setMessage(output)
		}
	} else {
		if err := c.compiler.Compile(c.Source); err != nil {
			c.setState(false)
			c.addError(err)
		}
	}
}
