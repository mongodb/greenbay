package check

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/blang/semver"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"

	"github.com/mongodb/grip"
)

func init() {
	name := "python-module-version"

	registry.AddJobType(name, func() amboy.Job {
		return &pythonModuleVersion{
			Base: NewBase(name, 0),
		}
	})
}

type pythonModuleVersion struct {
	Module            string `bson:"module" json:"module" yaml:"module"`
	Statement         string `bson:"statement" json:"statement" yaml:"statement"`
	Version           string `bson:"version" json:"version" yaml:"version"`
	MinVersion        string `bson:"minVersion" json:"minVersion" yaml:"minVersion"`
	MinRelationship   string `bson:"minRelationship" json:"minRelationship" yaml:"minRelationship"`
	PythonInterpreter string `bson:"python" json:"python" yaml:"python"`
	Relationship      string `bson:"relationship" json:"relationship" yaml:"relationship"`
	*Base             `bson:"metadata" json:"metadata" yaml:"metadata"`
}

func (c *pythonModuleVersion) validate() error {
	if c.PythonInterpreter == "" {
		// TODO: consider if we want to default to python2 on
		// some systems, or have a better default version.
		c.PythonInterpreter = "python"
		grip.Debug("no python interpreter specified, using default python from PATH")
	}

	switch c.Relationship {
	case "":
		grip.Debug("no relationship specified, using greater than or equal to (gte)")
		c.Relationship = "gte"
	case "gte", "lte", "lt", "gt", "eq":
		grip.Debugln("relationship for '%s' check set to '%s'", c.ID(), c.Relationship)
	default:
		return errors.Errorf("relationship '%s' for check '%s' is invalid",
			c.Relationship, c.ID())
	}

	switch c.MinRelationship {
	case "":
		grip.Debug("no relationship specified, using greater than or equal to (gte)")
		c.MinRelationship = "gte"
	case "gte", "lte", "lt", "gt", "eq":
		grip.Debugln("relationship for '%s' check set to '%s'", c.ID(), c.MinRelationship)
	default:
		return errors.Errorf("relationship '%s' for check '%s' is invalid",
			c.MinRelationship, c.ID())
	}

	return nil
}

func (c *pythonModuleVersion) Run() {
	c.startTask()

	defer c.MarkComplete()

	if err := c.validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	expected, err := semver.Parse(c.Version)
	if err != nil {
		c.setState(false)
		c.AddError(err)
		c.setMessage(fmt.Sprintf("could not parse expected version '%s'", c.Version))
		return
	}

	var minExpected semver.Version
	if c.MinVersion != "" {
		minExpected, err = semver.Parse(c.MinVersion)
		if err != nil {
			c.setState(false)
			c.AddError(err)
			c.setMessage(fmt.Sprintf("could not parse expected version '%s'", c.Version))
			return
		}
	}

	cmdArgs := []string{
		c.PythonInterpreter, "-c",
		fmt.Sprintf("import %s; print(%s)", c.Module, c.Statement),
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	versionOut, err := cmd.Output()
	version := strings.Trim(string(versionOut), "\r\t\n ")
	if err != nil {
		c.setState(false)
		c.AddError(err)
		c.setMessage(version)
		return
	}

	parsed, err := semver.Parse(version)
	if err != nil {
		c.setState(false)
		c.AddError(err)
		c.setMessage(fmt.Sprintf("could not parse version '%s' from module '%s'",
			version, c.Module))
		return
	}

	var result bool
	result, err = compareVersions(c.Relationship, parsed, expected)
	if err != nil {
		// this should be unreachable, because the earlier
		// validate will have caught it.
		c.setState(false)
		c.AddError(err)
		return
	}

	if c.MinVersion != "" {
		gteMin, err := compareVersions(c.MinRelationship, parsed, minExpected)
		if err != nil {
			c.setState(false)
			c.AddError(err)
			return
		}

		result = result && gteMin
	}

	if !result {
		c.setState(false)
		msg := fmt.Sprintln(parsed, c.Relationship, expected)
		c.AddError(errors.Errorf("check failed: %s", msg))
		c.setMessage(msg)
		return
	}

	c.setState(true)
}
