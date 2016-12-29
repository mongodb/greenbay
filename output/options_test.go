package output

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/greenbay/check"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type OptionsSuite struct {
	tmpDir  string
	opts    *Options
	require *require.Assertions
	queue   amboy.Queue
	cancel  context.CancelFunc
	suite.Suite
}

func TestOptionsSuite(t *testing.T) {
	suite.Run(t, new(OptionsSuite))
}

// Suite Fixtures:

func (s *OptionsSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.require = s.Require()

	tmpDir, err := ioutil.TempDir("", uuid.NewV4().String())
	s.require.NoError(err)
	s.tmpDir = tmpDir

	s.queue = queue.NewLocalUnordered(2)
	s.require.NoError(s.queue.Start(ctx))
	num := 5
	for i := 0; i < num; i++ {
		check := &mockCheck{Base: check.Base{Base: &job.Base{}}}
		check.SetID(fmt.Sprintf("mock-check-%d", i))
		s.NoError(s.queue.Put(check))
	}
	s.Equal(num, s.queue.Stats().Total)
	amboy.Wait(s.queue)
}

func (s *OptionsSuite) SetupTest() {
	s.opts = &Options{}
}

func (s *OptionsSuite) TearDownSuite() {
	s.NoError(os.RemoveAll(s.tmpDir))
	s.cancel()
}

// Test cases:

func (s *OptionsSuite) TestConstructorInvertsValueOfQuietArgument() {
	for _, q := range []bool{true, false} {
		opt, err := NewOptions("", "gotest", q)
		s.NoError(err)
		s.Equal(!q, opt.writeStdOut)
	}
}

func (s *OptionsSuite) TestEmptyFileNameDisablesWritingFiles() {
	opt, err := NewOptions("", "gotest", true)
	s.NoError(err)
	s.Equal("", opt.fn)
	s.False(opt.writeFile)
}

func (s *OptionsSuite) TestSpecifiedFileEnablesWritingFiles() {
	fn := filepath.Join(s.tmpDir, "enabled-one")
	opt, err := NewOptions(fn, "gotest", false)
	s.NoError(err)
	s.Equal(fn, opt.fn)
	s.True(opt.writeFile)
}

func (s *OptionsSuite) TestConstructorErrorsWithInvalidOutputFormats() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		opt, err := NewOptions("", format, true)
		s.Error(err)
		s.Nil(opt)
	}
}

func (s *OptionsSuite) TestResultsProducderGeneratorErrorsWithInvalidFormat() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		s.opts.format = format
		rp, err := s.opts.GetResultsProducer()
		s.Error(err)
		s.Nil(rp)
	}
}

func (s *OptionsSuite) TestResultsProducerOperationFailsWIthInvaildFormat() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		s.opts.format = format
		err := s.opts.ProduceResults(nil)
		s.Error(err)
	}
}

func (s *OptionsSuite) TestGetResultsProducerForValidFormats() {
	for _, format := range []string{"gotest", "result", "log"} {
		s.opts.format = format
		rp, err := s.opts.GetResultsProducer()
		s.NoError(err)
		s.NotNil(rp)
		s.Implements((*ResultsProducer)(nil), rp)
	}
}

func (s *OptionsSuite) TestResultsProducerOperationReturnsErrorWithNilQueue() {
	for _, format := range []string{"gotest", "result", "log"} {
		opt, err := NewOptions("", format, true)
		s.NoError(err)

		s.Error(opt.ProduceResults(nil))
	}
}

func (s *OptionsSuite) TestResultsToStandardOutButNotPrint() {
	for _, format := range []string{"gotest", "result", "log"} {
		opt, err := NewOptions("", format, true)
		s.NoError(err)

		s.NoError(opt.ProduceResults(s.queue))
	}
}

func (s *OptionsSuite) TestResultsToFileOnly() {
	for idx, format := range []string{"gotest", "result", "log"} {
		fn := filepath.Join(s.tmpDir, fmt.Sprintf("enabled-two-%d", idx))
		opt, err := NewOptions(fn, format, false)

		s.NoError(err)
		s.NoError(opt.ProduceResults(s.queue))
	}
}

func (s *OptionsSuite) TestResultsToFileAndOutput() {
	for idx, format := range []string{"gotest", "result", "log"} {
		fn := filepath.Join(s.tmpDir, fmt.Sprintf("enabled-three-%d", idx))
		opt, err := NewOptions(fn, format, true)

		s.NoError(err)
		s.NoError(opt.ProduceResults(s.queue))
	}
}
