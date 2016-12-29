package output

import (
	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// Options represents all operations for output generation, and
// provides methods for accessing and producing results using that
// configuration regardless of underlying output format.
type Options struct {
	writeFile   bool
	writeStdOut bool
	fn          string
	format      string
}

// NewOptions provides a constructor to generate a valid Options
// structure. Returns an error if the specified format is not valid or
// registered.
func NewOptions(fn, format string, quiet bool) (*Options, error) {
	_, exists := GetResultsFactory(format)
	if !exists {
		return nil, errors.Errorf("no results format named '%s' exists", format)
	}

	o := &Options{}
	o.format = format
	o.writeStdOut = !quiet

	if fn != "" {
		o.writeFile = true
		o.fn = fn
	}

	return o, nil
}

// GetResultsProducer returns the ResultsProducer implementation
// specified in the Options structure, and returns an error if the
// format specified in the structure does not refer to a registered
// type.
func (o *Options) GetResultsProducer() (ResultsProducer, error) {
	factory, ok := GetResultsFactory(o.format)
	if !ok {
		return nil, errors.Errorf("no results format named '%s' exists", o.format)
	}

	rp := factory()

	return rp, nil
}

// ProduceResults takes an amboy.Queue object and produces results
// according to the options specified in the Options
// structure. ProduceResults returns an error if any of the tests
// failed in the operation.
func (o *Options) ProduceResults(q amboy.Queue) error {
	if q == nil {
		return errors.New("cannot populate results with a nil queue")
	}

	return o.CollectResults(q.Results())
}

// CollectResults takes a channel that produces jobs and produces results
// according to the options specified in the Options
// structure. ProduceResults returns an error if any of the tests
// failed in the operation.
func (o *Options) CollectResults(jobs <-chan amboy.Job) error {
	rp, err := o.GetResultsProducer()
	if err != nil {
		return errors.Wrap(err, "problem fetching results producer")
	}

	if err := rp.Populate(jobs); err != nil {
		return errors.Wrap(err, "problem generating results content")
	}

	// Actually write output to respective streems
	catcher := grip.NewCatcher()

	if o.writeStdOut {
		catcher.Add(rp.Print())
	}

	if o.writeFile {
		catcher.Add(rp.ToFile(o.fn))
	}

	return catcher.Resolve()
}
