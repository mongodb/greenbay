/*
Package operations provides the core greenbay application
functionality, distinct from user interfaces.

The core greenbay test execution code is available here to support
better testing and alternate interfaces. Currently the only interface
is a command line interface, but we could wrap this functionality in a
web service to support easier integration with monitoring tools or
other health-check services.

The core functionality of the application is in the GreenbayApp
structure which stores application and facilitates the integration of
output production, test running, and test configuration.
*/
package operations

import (
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"golang.org/x/net/context"
)

// GreenbayApp encapsulates the execution of a greenbay run. You can
// construct the object, either with NewApp(), or by building a
// GreenbayApp structure yourself.
type GreenbayApp struct {
	Output     *output.Options
	Conf       *config.GreenbayTestConfig
	NumWorkers int
	Tests      []string
	Suites     []string
}

// NewApp configures the greenbay application and manages the
// construction of the main config object as well as the output
// configuration structure. Returns an error if there are problems
// constructing either the main config or the output
// configuration objects.
func NewApp(confPath, outFn, format string, quiet bool, jobs int, suite, tests []string) (*GreenbayApp, error) {
	out, err := output.NewOptions(outFn, format, quiet)
	if err != nil {
		return nil, errors.Wrap(err, "problem generating output definition")
	}

	conf, err := config.ReadConfig(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "problem parsing config file")
	}

	app := &GreenbayApp{
		Conf:       conf,
		Output:     out,
		NumWorkers: jobs,
		Tests:      tests,
		Suites:     suite,
	}

	return app, nil
}

// Run executes all tasks defined in the application, and produces
// results as described by the output configuration. Returns an error
// if any test failed and/or if there were any problems with test
// execution.
func (a *GreenbayApp) Run(ctx context.Context) error {
	if a.Conf == nil || a.Output == nil {
		return errors.New("GreenbayApp is not correctly constructed:" +
			"system and output configuration must be specified.")
	}

	// make sure we clean up after ourselves if we return early
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := queue.NewLocalUnordered(a.NumWorkers)

	if err := q.Start(ctx); err != nil {
		return errors.Wrap(err, "problem starting workers")
	}

	// begin "real" work
	start := time.Now()
	catcher := grip.NewCatcher()

	for check := range a.Conf.GetAllTests(a.Tests, a.Suites) {
		if check.Err != nil {
			catcher.Add(check.Err)
			continue
		}
		catcher.Add(q.Put(check.Job))
	}
	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "problem collecting and submitting jobs")
	}

	stats := q.Stats()
	grip.Noticef("registered %d jobs, running checks now", stats.Total)
	amboy.WaitCtxInterval(ctx, q, 10*time.Millisecond)

	grip.Noticef("checks complete in [num=%d, runtime=%s] ", stats.Total, time.Since(start))
	if err := a.Output.ProduceResults(q); err != nil {
		return errors.Wrap(err, "problems encountered during tests")
	}

	return nil
}
