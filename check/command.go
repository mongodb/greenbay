package check

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func init() {
	for name, shouldFail := range map[string]bool{
		"shell-operation":       false,
		"shell-operation-error": true,
	} {
		jobName := name
		registry.AddJobType(jobName, func() amboy.Job {
			return &shellOperation{
				Environment: make(map[string]string),
				shouldFail:  shouldFail,
				Base:        NewBase(jobName, 0), // (name, version)
			}
		})
	}
}

type shellOperation struct {
	Command          string            `bson:"command" json:"command" yaml:"command"`
	WorkingDirectory string            `bson:"working_directory" json:"working_directory" yaml:"working_directory"`
	Environment      map[string]string `bson:"environment" json:"environment" yaml:"environment"`
	shouldFail       bool

	*Base `bson:"base" json:"base" yaml:"base"`
}

func (c *shellOperation) Run() {
	c.startTask()
	defer c.markComplete()

	logMsg := []string{fmt.Sprintf("command='%s'", c.Command)}

	// I don't like "sh -c" as a thing, but it parallels the way
	// that Evergreen runs tasks (for now,) and it gets us away
	// from needing to do special shlex parsing, though
	// (https://github.com/google/shlex) seems like a good start.
	cmd := exec.Command("sh", "-c", c.Command)
	if c.WorkingDirectory != "" {
		cmd.Dir = c.WorkingDirectory
		logMsg = append(logMsg, fmt.Sprintf("dir='%s'", c.WorkingDirectory))
	}

	if len(c.Environment) > 0 {
		env := []string{}
		for key, value := range c.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
		logMsg = append(logMsg, fmt.Sprintf("env='%s'", strings.Join(env, " ")))
	}

	c.setState(true) // default to pass
	out, err := cmd.CombinedOutput()
	if err != nil {
		logMsg = append(logMsg, fmt.Sprintf("err='%+v'", err))

		c.setMessage(string(out))

		if !c.shouldFail {
			c.setState(false)
			c.addError(errors.Wrapf(err, "command failed",
				c.ID(), c.Command))
		}
	} else if c.shouldFail {
		c.setState(false)
		c.addError(errors.Errorf("command '%s' succeeded but test expects it to fail",
			c.Command))
	}

	grip.Debug(strings.Join(logMsg, ", "))

	if !c.getState() {
		c.setMessage(string(out))
	}
}
