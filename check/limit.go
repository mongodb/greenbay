package check

import (
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

func registerSystemLimitChecks() {
	limitCheckFactoryFactory := func(name string, cfunc limitValueCheck) func() amboy.Job {
		return func() amboy.Job {
			return &limitCheck{
				Base:      NewBase(name, 0),
				limitTest: cfunc,
			}
		}
	}

	for name, checkFunc := range limitValueCheckTable() {
		registry.AddJobType(name, limitCheckFactoryFactory(name, checkFunc))
	}
}

type limitValueCheck func(int) (bool, error)

type limitCheck struct {
	Value     int `bson:"value" json:"value" yaml:"value"`
	*Base     `bson:"metadata" json:"metadata" yaml:"metadata"`
	limitTest limitValueCheck
}

func (c *limitCheck) Run() {
	c.startTask()
	defer c.markComplete()

	c.setState(true) // default to true unless proven otherwise.

	result, err := c.limitTest(c.Value)
	if err != nil {
		c.setState(false)
		c.addError(err)
		return
	}

	if !result {
		c.setState(false)
		c.addError(errors.Errorf("limit in check %s is incorrect", c.ID()))
		return
	}
}
