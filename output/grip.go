package output

import (
	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// GripOutput provides a ResultsProducer implementation that writes
// the results of a greenbay run to logging using the grip logging
// package.
type GripOutput struct {
	passedMsgs []message.Composer
	failedMsgs []message.Composer
}

// Populate generates output messages based on the content (via the
// Results() method) of an amboy.Queue instance. All jobs processed by
// that queue must also implement the greenbay.Checker
// interface. Returns an error if there are any invalid jobs.
func (r *GripOutput) Populate(queue amboy.Queue) error {
	if queue == nil {
		return errors.New("cannot populate results with a nil queue")
	}

	catcher := grip.NewCatcher()
	for wu := range jobsToCheck(queue.Results()) {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		dur := wu.output.Timing.Start.Sub(wu.output.Timing.End)
		if wu.output.Passed {
			r.passedMsgs = append(r.passedMsgs,
				message.NewFormatedMessage("PASSED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		} else {
			r.failedMsgs = append(r.passedMsgs,
				message.NewFormatedMessage("FAILED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		}
	}

	return catcher.Resolve()
}

// ToFile logs, to the specified file, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *GripOutput) ToFile(fn string) error {
	logger := grip.NewJournaler("greenbay")
	if err := logger.UseFileLogger(fn); err != nil {
		return errors.Wrapf(err, "problem setting up output logger to file '%s'", fn)
	}

	logger.SetDefaultLevel(level.Info)
	logger.SetThreshold(level.Info)

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

// Print logs, to standard output, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *GripOutput) Print() error {
	logger := grip.NewJournaler("greenbay")
	if err := logger.UseNativeLogger(); err != nil {
		return errors.Wrap(err, "problem setting up logger")
	}

	logger.SetDefaultLevel(level.Info)
	logger.SetThreshold(level.Info)

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

func (r *GripOutput) logResults(logger grip.Journaler) {
	for _, msg := range r.passedMsgs {
		logger.Notice(msg)
	}

	for _, msg := range r.failedMsgs {
		logger.Alert(msg)
	}
}
