package output

import (
	"github.com/mongodb/amboy"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
)

type workUnit struct {
	output greenbay.CheckOutput
	err    error
}

// jobsToCheck converts a channel of amboy.Job objects to
// greenbay.Checker interface. If a job object is not able to be
// converted to greenbay.Checker, this operation panics.
func jobsToCheck(jobs <-chan amboy.Job) <-chan workUnit {
	output := make(chan workUnit)

	go func() {
		for j := range jobs {
			c, err := convert(j)
			if err != nil {
				output <- workUnit{
					output: greenbay.CheckOutput{},
					err:    err,
				}
				continue
			}

			output <- workUnit{
				output: c.Output(),
				err:    nil,
			}
		}
		close(output)
	}()

	return output
}

func convert(j amboy.Job) (greenbay.Checker, error) {
	c, ok := j.(greenbay.Checker)
	if ok {
		return c, nil
	}

	err := errors.Errorf("job %s (%T) does not implement greenbay.Checker interface",
		j.ID(), j)

	return nil, err
}
