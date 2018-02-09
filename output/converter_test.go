package output

import (
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/greenbay"
	"github.com/mongodb/greenbay/check"
	"github.com/stretchr/testify/assert"
)

type mockCheck struct {
	hasRun bool
	check.Base
}

func (c *mockCheck) Run() {
	c.Base.WasSuccessful = true
	c.Base.MarkComplete()
	c.hasRun = true
}

func TestConverter(t *testing.T) {
	assert := assert.New(t)

	j := job.NewShellJob("echo foo", "")
	assert.NotNil(j)
	c, err := convert(j)
	assert.Error(err)
	assert.Nil(c)

	mc := &mockCheck{Base: check.Base{Base: &job.Base{}}}
	assert.Implements((*amboy.Job)(nil), mc)
	assert.Implements((*greenbay.Checker)(nil), mc)

	c, err = convert(mc)
	assert.NoError(err)
	assert.NotNil(c)
}

func TestJobToCheckGenerator(t *testing.T) {
	assert := assert.New(t) // nolint
	input := make(chan amboy.Job)
	output := jobsToCheck(false, input)

	i := &mockCheck{Base: check.Base{Base: &job.Base{}}}
	assert.Implements((*amboy.Job)(nil), i)
	input <- i

	o := <-output
	assert.NoError(o.err)
	assert.IsType(greenbay.CheckOutput{}, o.output)

	close(input)
}
