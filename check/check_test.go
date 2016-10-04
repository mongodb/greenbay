package check

import (
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CheckSuite struct {
	name    string
	factory registry.JobFactory
	check   greenbay.Checker
	require *require.Assertions
	suite.Suite
}

// Test constructors. For every new check, you should register a new
// version of the suite, specifying a different "name" value.

func TestMockCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "mock-check"
	suite.Run(t, s)
}

// Test Fixtures

func (s *CheckSuite) SetupSuite() {
	s.require = s.Require()
	factory, err := registry.GetJobFactory(s.name)
	s.NoError(err)

	s.factory = factory
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

	if output.Passed {
		s.NoError(err)
	} else {
		s.Error(err)
	}
}
