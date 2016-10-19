package config

import (
	"encoding/json"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
)

// GreenbayTestConfig defines the structure for a single greenbay test
// run, including execution behavior (options) and check definitions.
type GreenbayTestConfig struct {
	Options  *options             `bson:"options" json:"options" yaml:"options"`
	RawTests []rawTest            `bson:"tests" json:"tests" yaml:"tests"`
	tests    map[string]amboy.Job // maping of test names to test objects
	suites   map[string][]string  // mapping of suite names to test names
	mutex    sync.RWMutex
}

type options struct {
	ContineOnError bool   `bson:"continue_on_error" json:"continue_on_error" yaml:"continue_on_error"`
	ReportFormat   string `bson:"report_format" json:"report_format" yaml:"report_format"`
	Jobs           int    `bson:"jobs" json:"jobs" yaml:"jobs"` // number of job workers.
}

func newTestConfig() *GreenbayTestConfig {
	conf := &GreenbayTestConfig{}
	conf.reset()
	conf.Options.Jobs = runtime.NumCPU()

	return conf
}

func (c *GreenbayTestConfig) reset() {
	c.suites = make(map[string][]string)
	c.tests = make(map[string]amboy.Job)
}

// ReadConfig takes a path name to a configuration file (yaml
// formatted,) and returns a configuration format.
func ReadConfig(fn string) (*GreenbayTestConfig, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem reading greenbay config file: %s", fn)
	}

	format, err := getFormat(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem determining format of file %s", fn)
	}

	// Parse data:
	data, err = getJSONFormattedConfig(format, data)
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing config from file %s", fn)
	}

	c := newTestConfig()
	// we don't take the lock here because this function doesn't
	// spawn threads, and nothing else can see the object we're
	// building. If either of those things change we should take
	// the lock here.

	// now we have a json formated byte slice in data and we can
	// unmarshal it as we want.
	err = json.Unmarshal(data, c)
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing config: %s", fn)
	}

	err = c.parseTests()
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing tests from file: %s", fn)
	}
	return c, nil
}

// JobWithError is a type used by the test generators and contains an
// amboy.Job and an error message.
type JobWithError struct {
	Job amboy.Job
	Err error
}

// TestsForSuites takes the name of a suite and then produces a sequence of
// jobs that are part of that suite.
func (c *GreenbayTestConfig) TestsForSuites(names ...string) <-chan JobWithError {
	output := make(chan JobWithError)
	go func() {
		c.mutex.RLock()
		defer c.mutex.RUnlock()

		seen := make(map[string]struct{})
		for _, suite := range names {
			tests, ok := c.suites[suite]
			if !ok {
				output <- JobWithError{
					Job: nil,
					Err: errors.Errorf("suite named '%s' does not exist", suite),
				}

				continue
			}

			for _, test := range tests {
				j, ok := c.tests[test]

				var err error
				if !ok {
					err = errors.Errorf("test name %s is specified in suite %s"+
						"but does not exist", test, suite)
				}

				if _, ok := seen[test]; ok {
					// this means a test is specified in more than one suite,
					// and we only want to dispatch it once.
					continue
				}

				seen[test] = struct{}{}

				if err != nil {
					output <- JobWithError{Job: nil, Err: err}
					continue
				}

				output <- JobWithError{Job: j, Err: nil}
			}
		}

		close(output)
	}()

	return output
}

// TestsByName is a generator takes one or more names of tests (as
// strings) and returns a channel of result objects that contain
// errors (if those names do not exist) and job objects.
func (c *GreenbayTestConfig) TestsByName(names ...string) <-chan JobWithError {
	output := make(chan JobWithError)
	go func() {
		c.mutex.RLock()
		defer c.mutex.RUnlock()

		for _, test := range names {
			j, ok := c.tests[test]

			if !ok {
				output <- JobWithError{
					Job: nil,
					Err: errors.Errorf("no test named %s", test),
				}
				continue
			}

			output <- JobWithError{Job: j, Err: nil}
		}

		close(output)
	}()

	return output
}
