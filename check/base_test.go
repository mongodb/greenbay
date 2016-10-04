package check

import (
	"errors"
	"strings"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BaseCheckSuite struct {
	base    *Base
	require *require.Assertions
	suite.Suite
}

func TestBaseCheckSuite(t *testing.T) {
	suite.Run(t, new(BaseCheckSuite))
}

func (s *BaseCheckSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *BaseCheckSuite) SetupTest() {
	s.base = &Base{dep: dependency.NewAlways()}
}

func (s *BaseCheckSuite) TestInitialValuesOfBaseObject() {
	s.False(s.base.IsComplete)
	s.False(s.base.WasSuccessful)
	s.Len(s.base.Errors, 0)
}

func (s *BaseCheckSuite) TestAddErrorWithNilObjectDoesNotChangeErrorState() {
	for i := 0; i < 100; i++ {
		s.base.addError(nil)
		s.NoError(s.base.Error())
		s.Len(s.base.Errors, 0)
		s.False(s.base.hasErrors())
	}
}

func (s *BaseCheckSuite) TestAddErrorsPersisstsErrorsInJob() {
	for i := 1; i <= 100; i++ {
		s.base.addError(errors.New("foo"))
		s.Error(s.base.Error())
		s.Len(s.base.Errors, i)
		s.True(s.base.hasErrors())
		s.Len(strings.Split(s.base.Error().Error(), "\n"), i)
	}
}

func (s *BaseCheckSuite) TestIdIsAccessorForTaskIDAttribute() {
	s.Equal(s.base.TaskID, s.base.ID())
	s.base.TaskID = "foo"
	s.Equal("foo", s.base.ID())
	s.Equal(s.base.TaskID, s.base.ID())
}

func (s *BaseCheckSuite) TestDependencyAccessorIsCorrect() {
	s.Equal(s.base.dep, s.base.Dependency())
	s.base.SetDependency(dependency.NewAlways())
	s.Equal(dependency.AlwaysRun, s.base.Dependency().Type().Name)
}

func (s *BaseCheckSuite) TestSetDependencyAccepstAndPerisstsChangesToDependencyType() {
	s.Equal(dependency.AlwaysRun, s.base.dep.Type().Name)
	localDep := dependency.NewLocalFileInstance()
	s.NotEqual(localDep.Type().Name, dependency.AlwaysRun)
	s.base.SetDependency(localDep)
	s.Equal(dependency.LocalFileRelationship, s.base.dep.Type().Name)
}

func (s *BaseCheckSuite) TestOutputStructGenertedReflectsStateOfBaseObject() {
	s.base = &Base{
		TaskID: "foo",
		JobType: amboy.JobType{
			Name:    "base-greenbay-check",
			Version: 42,
			Format:  amboy.JSON,
		},
		TestSuites:    []string{"foo", "bar"},
		IsComplete:    true,
		WasSuccessful: true,
		Errors:        []error{errors.New("foo")},
		Message:       "baz",
	}

	output := s.base.Output()
	s.Equal("foo", output.Name)
	s.Equal("base-greenbay-check", output.Check)
	s.Equal("foo", output.Suites[0])
	s.Equal("bar", output.Suites[1])
	s.True(output.Completed)
	s.True(output.Passed)
	s.Equal("foo", output.Error)
	s.Equal("baz", output.Message)
}

func (s *BaseCheckSuite) TestMarkCompleteHelperSetsCompleteState() {
	s.False(s.base.IsComplete)
	s.False(s.base.Completed())
	s.base.markComplete()

	s.True(s.base.IsComplete)
	s.True(s.base.Completed())
}

func (s *BaseCheckSuite) TestMutableIDMethod() {
	for _, name := range []string{"foo", "bar", "baz", "bot"} {
		s.base.SetID(name)
		s.NotEqual(s.base.Name(), s.base.ID())
		s.Equal(name, s.base.ID())
		s.Equal(s.base.ID(), s.base.TaskID)
	}
}

func (s *BaseCheckSuite) TestStatMutability() {
	for _, state := range []bool{true, false, false, true, true} {
		s.base.setState(state)
		s.Equal(state, s.base.WasSuccessful)
	}
}

func (s *BaseCheckSuite) TestSetMessageConvertsTypesToString() {
	var mOne interface{}
	mOne = "foo"
	s.base.setMessage(mOne)
	s.Equal("foo", s.base.Message)

	s.base.setMessage(true)
	s.Equal("true", s.base.Message)

	s.base.setMessage(nil)
	s.Equal("<nil>", s.base.Message)

	s.base.setMessage(100)
	s.Equal("100", s.base.Message)

	s.base.setMessage(112)
	s.Equal("112", s.base.Message)

	s.base.setMessage(errors.New("foo"))
	s.Equal("foo", s.base.Message)

	s.base.setMessage(errors.New("bar"))
	s.Equal("bar", s.base.Message)

	strs := []string{"foo", "bar", "baz"}
	s.base.setMessage([]string{"foo", "bar", "baz"})
	s.Equal(strings.Join(strs, "\n"), s.base.Message)
}

func (s *BaseCheckSuite) TestSetSuitesOverriedsExistingSuites() {
	cases := [][]string{
		[]string{},
		[]string{"foo", "bar"},
		[]string{"1", "false"},
		[]string{"greenbay", "kenosha", "jainseville"},
	}

	for _, suites := range cases {
		s.base.SetSuites(suites)
		s.Equal(suites, s.base.Suites())
	}
}

func (s *BaseCheckSuite) TestRoutTripAbilityThroughImportAndExport() {
	s.base.JobType = amboy.JobType{
		Format:  amboy.JSON,
		Version: 42,
	}

	s.Equal(42, s.base.JobType.Version)
	out, err := s.base.Export()

	s.NoError(err)
	s.NotNil(out)

	s.base.JobType.Version = 21
	s.Equal(21, s.base.JobType.Version)

	err = s.base.Import(out)
	s.NoError(err)
	s.Equal(42, s.base.JobType.Version)

}
