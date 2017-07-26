package output

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/mongodb/amboy"
	"github.com/mongodb/greenbay"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Report implements a single machine-parsable json format for results, for use in the rest API
type Report struct {
	results   map[string]*greenbay.CheckOutput
	hasErrors bool
}

func (r *Report) Populate(jobs <-chan amboy.Job) error {
	r.results = make(map[string]*greenbay.CheckOutput)
	catcher := grip.NewCatcher()

	for check := range jobsToCheck(jobs) {
		if check.err != nil {
			r.hasErrors = true
			catcher.Add(check.err)
			continue
		}
		r.results[check.output.Name] = &check.output
		if !check.output.Passed {
			r.hasErrors = true
		}
	}

	return catcher.Resolve()
}

func (r *Report) ToFile(fn string) error {
	data, err := r.getJSON()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := ioutil.WriteFile(fn, data, 0644); err != nil {
		return errors.Wrapf(err, "problem writing output to %s", fn)
	}

	if r.hasErrors {
		return errors.New("tests failed")
	}

	return nil
}

func (r *Report) Print() error {
	data, err := r.getJSON()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println(string(data))

	if r.hasErrors {
		return errors.New("tests failed")
	}

	return nil
}

func (r *Report) getJSON() ([]byte, error) {
	data, err := json.MarshalIndent(r.results, "", "   ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "problem marhsaling results")
	}
	return data, nil
}
