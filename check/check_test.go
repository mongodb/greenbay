package check

import (
	"fmt"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

type CheckSuite struct {
	name    string
	factory registry.JobFactory
	check   greenbay.Checker
	require *require.Assertions
	suite.Suite
}

func TestCheckSuite(t *testing.T) {
	for name := range registry.JobTypeNames() {
		suite.Run(t, &CheckSuite{name: name})
	}
}

// Test Fixtures

func (s *CheckSuite) SetupSuite() {
	s.require = s.Require()
	factory, err := registry.GetJobFactory(s.name)
	s.NoError(err)

	s.factory = factory
	grip.Infoln("running check suite for", s.name)
}

func (s *CheckSuite) SetupTest() {
	s.require.NotNil(s.factory)
	s.check = s.factory().(greenbay.Checker)
	s.require.NotNil(s.check)
}

// Test Cases

func (s *CheckSuite) TestCheckImplementsRequiredInterface() {
	s.Implements((*amboy.Job)(nil), s.check)
	s.Implements((*greenbay.Checker)(nil), s.check)
}

func (s *CheckSuite) TestInitialStateHasCorrectDefaults() {
	output := s.check.Output()
	s.False(output.Completed)
	s.False(output.Passed)
	s.False(s.check.Completed())
	s.NoError(s.check.Error())
	s.Equal("", output.Error)
	s.Equal(s.name, output.Check)
	s.Equal(s.name, s.check.Type().Name)
}

func (s *CheckSuite) TestRunningTestsHasImpact() {
	output := s.check.Output()
	s.False(output.Completed)
	s.False(s.check.Completed())
	s.False(output.Passed)

	s.check.Run()

	output = s.check.Output()
	s.True(output.Completed)
	s.True(s.check.Completed())
}

func (s *CheckSuite) TestFailedChecksShouldReturnErrors() {
	s.check.Run()
	output := s.check.Output()
	s.True(s.check.Completed())

	err := s.check.Error()

	msg := fmt.Sprintf("%T: %+v", s.check, output)
	if output.Passed {
		s.NoError(err, msg)
	} else {
		s.Error(err, msg)
	}
}
