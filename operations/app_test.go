package operations

import (
	"testing"

	"github.com/mongodb/greenbay/check"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type AppSuite struct {
	app *GreenbayApp
	suite.Suite
}

func TestAppSuite(t *testing.T) {
	suite.Run(t, new(AppSuite))
}

func (s *AppSuite) SetupSuite() {

}

func (s *AppSuite) SetupTest() {
	s.app = &GreenbayApp{}
}

type mockCheck struct {
	hasRun bool
	check.Base
}

func (c *mockCheck) Run() {
	c.Base.WasSuccessful = true
	c.Base.IsComplete = true
	c.hasRun = true
}

// Test cases:

func (s *AppSuite) TestRunFailsWithUninitailizedConfAndOrOutput() {
	ctx := context.Background()
	s.Nil(s.app.Conf)
	s.Nil(s.app.Output)
	s.Error(s.app.Run(ctx))

	conf := &config.GreenbayTestConfig{}
	s.NotNil(conf)
	s.app.Conf = conf
	s.NotNil(s.app.Conf)
	s.Nil(s.app.Output)
	s.Error(s.app.Run(ctx))

	s.app.Conf = nil

	out := &output.Options{}
	s.NotNil(out)
	s.app.Output = out
	s.NotNil(s.app.Output)
	s.Nil(s.app.Conf)
	s.Error(s.app.Run(ctx))
}

func (s *AppSuite) TestConsturctorFailsIfConfPathDoesNotExist() {
	app, err := NewApp("DOES-NOT-EXIST", "", "gotest", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

func (s *AppSuite) TestConsturctorFailsWithEmptyConfPath() {
	app, err := NewApp("", "", "gotest", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

func (s *AppSuite) TestConstructorFailsWithInvalidOutputConfigurations() {
	app, err := NewApp("", "", "DOES-NOT-EXIST", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

// TODO: add tests that exercise successful runs and dispatch actual
// tests and suites,but to do this we'll want to have better mock
// tests and configs, so holding off on that until MAKE-101
