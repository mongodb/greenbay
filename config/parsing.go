package config

import (
	"encoding/json"
	"path/filepath"

	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"gopkg.in/yaml.v2"
)

// Helper functions that convert yaml-to-json so that the constructor
// can just convert to the required struct type.

func getFormat(fn string) (amboy.Format, error) {
	ext := filepath.Ext(fn)

	if ext == ".yaml" || ext == ".yml" {
		return amboy.YAML, nil
	} else if ext == ".json" {
		return amboy.JSON, nil
	}

	return -1, errors.Errorf("greenbay does not support files with '%s' extension", ext)
}

func getJSONFormattedConfig(format amboy.Format, data []byte) ([]byte, error) {
	if format == amboy.JSON {
		return data, nil
	} else if format == amboy.YAML {
		// the yaml package does not include a way to do the kind of
		// delayed parsing that encoding/json permits, so we cycle
		// into a map and then through the JSON parser itself.
		intermediateOut := make(map[string]interface{})

		err := yaml.Unmarshal(data, intermediateOut)
		if err != nil {
			return nil, errors.Wrap(err, "problem parsing yaml config")
		}

		data, err = json.Marshal(intermediateOut)
		if err != nil {
			// this requires valid yaml that isn't also
			// valid json, which should be possible, but
			// isn't exceptionally likely. worth catching.
			return nil, errors.Wrap(err, "problem converting yaml to intermediate json")
		}

		return data, nil
	}

	return nil, errors.Errorf("format %s is not supported", format)
}

////////////////////////////////////////////////////////////////////////
//
// Internal Methods used by the constructor (ReadConfig) function.
//
////////////////////////////////////////////////////////////////////////

func (c *GreenbayTestConfig) parseTests() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	catcher := grip.NewCatcher()
	for _, msg := range c.RawTests {
		c.addSuites(msg.Name, msg.Suites)

		testJob, err := msg.resolveCheck()
		if err != nil {
			catcher.Add(errors.Wrapf(err, "problem resolving %s", msg.Name))
			continue
		}

		err = c.addTest(msg.Name, testJob)
		if err != nil {
			grip.Alert(err)
			catcher.Add(err)
		}
	}

	return catcher.Resolve()
}

// These methods are unsafe, and need to be used within the context a lock.

func (c *GreenbayTestConfig) addSuites(name string, suites []string) {
	for _, suite := range suites {
		if _, ok := c.suites[suite]; !ok {
			c.suites[suite] = []string{}
		}

		c.suites[suite] = append(c.suites[suite], name)
	}
}

func (c *GreenbayTestConfig) addTest(name string, j amboy.Job) error {
	if _, ok := c.tests[name]; ok {
		return errors.Errorf("two tests named '%s'", name)
	}

	c.tests[name] = j
	grip.Infoln("added test named:", name)

	return nil
}

// end unsafe methods
