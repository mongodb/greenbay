package operations

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/amboy/rest"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/tychoish/gimlet"
)

// GreenbayService holds the configuration and operations for running
// a Greenbay service.
type GreenbayService struct {
	DisableStats bool
	service      *rest.Service
	conf         *config.GreenbayTestConfig
	output       *output.Options
}

// NewService constructs a GreenbayService, but does not start the
// service. You will need to run Open to start the underlying workers and
// Run to start the HTTP service. You can set the host to the empty
// string, to bind the service to all interfaces.
func NewService(confPath string, host string, port int) (*GreenbayService, error) {
	s := &GreenbayService{
		// this operation loads all job instance names from
		// greenbay and and constructs the amboy.rest.Service object.
		service: rest.NewService(),
	}

	if confPath != "" {
		conf, err := config.ReadConfig(confPath)
		if err != nil {
			return nil, errors.Wrap(err, "problem parsing config file")
		}
		s.conf = conf
		s.output = &output.Options{}
	}

	app := s.service.App()

	if err := app.SetPort(port); err != nil {
		return nil, errors.Wrap(err, "problem constructing greenbay service")
	}

	if err := app.SetHost(host); err != nil {
		return nil, errors.Wrap(err, "problem constructing greenbay service")
	}

	return s, nil
}

// Open starts the service, using the configuration structure from the
// amboy.rest package to set the queue size, number of workers, and
// timeout when restarting the service.
func (s *GreenbayService) Open(ctx context.Context, info rest.ServiceInfo) error {
	app := s.service.App()

	if !s.DisableStats {
		grip.Info("registering endpoints for metrics reporting")
		app.AddRoute("/stats/system_info").Version(1).Get().Handler(s.sysInfoHandler)
		app.AddRoute("/stats/process_info/{pid:[0-9]+}").Version(1).Get().Handler(s.processInfoHandler)
		app.AddRoute("/stats/process_info").Version(1).Get().Handler(s.processInfoHandler)
	}

	if s.conf != nil {
		app.AddRoute("/check/reload").Version(1).Get().Handler(s.reloadConfig)
		app.AddRoute("/check/suite/{suite_id}").Version(1).Get().Handler(s.runSuiteHandler)
		app.AddRoute("/check/test/{test_id}").Version(1).Get().Handler(s.runTestHandler)
	}

	if err := s.service.OpenInfo(ctx, info); err != nil {
		return errors.Wrap(err, "problem opening queue")
	}

	return nil
}

// Close wraps the Close method from amboy.rest.Service, and releases
// all resources used by the queue.
func (s *GreenbayService) Close() {
	s.service.Close()
}

// Run wraps the Run method from amboy.rest.Service, and is responsible for
// starting the service. This method blocks until the service terminates.
func (s *GreenbayService) Run() {
	grip.CatchAlert(s.service.App().Run())
}

////////////////////////////////////////////////////////////////////////
//
// Handlers for adhoc job reporting
//
////////////////////////////////////////////////////////////////////////

func (s *GreenbayService) reloadConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.conf.Reload(); err != nil {
		gimlet.WriteErrorJSON(w, map[string]string{"error": err.Error()})
		return
	}

	gimlet.WriteJSON(w, map[string]string{"status": "config reloaded"})
}

func (s *GreenbayService) runSuiteHandler(w http.ResponseWriter, r *http.Request) {
	output, err := s.runAdhocTests(s.conf.TestsForSuites(gimlet.GetVars(r)["suite_id"]))

	if err != nil {
		gimlet.WriteErrorJSON(w, map[string]string{"error": err.Error()})
		return
	}

	gimlet.WriteJSON(w, output)
}

func (s *GreenbayService) runTestHandler(w http.ResponseWriter, r *http.Request) {
	output, err := s.runAdhocTests(s.conf.TestsByName(gimlet.GetVars(r)["test_id"]))

	if err != nil {
		gimlet.WriteErrorJSON(w, map[string]string{"error": err.Error()})
		return
	}

	gimlet.WriteJSON(w, output)
}

func (s *GreenbayService) runAdhocTests(jobs <-chan config.JobWithError) (interface{}, error) {
	catcher := grip.NewCatcher()
	q := queue.NewLocalUnordered(2)
	defer q.Runner().Close()

	for unit := range jobs {
		if unit.Err != nil {
			catcher.Add(unit.Err)
			continue
		}

		catcher.Add(q.Put(unit.Job))
	}

	if catcher.HasErrors() {
		return nil, catcher.Resolve()
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	amboy.WaitCtxInterval(ctx, q, 10*time.Millisecond)
	if ctx.Err() != nil {
		return nil, errors.New("check operation timedout")
	}

	output, err := s.output.Report(q.Results(ctx))
	if err != nil {
		return nil, err
	}

	return output, nil
}

////////////////////////////////////////////////////////////////////////
//
// Handlers for the Status Reporting Endpoints
//
////////////////////////////////////////////////////////////////////////

type statsErrorResponse struct {
	Pid   int    `json:"pid,omitempty"`
	Error string `json:"error"`
}

func (s *GreenbayService) sysInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := message.CollectSystemInfo()
	if !info.Loggable() {
		resp := &statsErrorResponse{Error: strings.Join(info.(*message.SystemInfo).Errors, "; ")}
		gimlet.WriteInternalErrorJSON(w, resp)
		return
	}

	gimlet.WriteJSON(w, info)
}

func (s *GreenbayService) processInfoHandler(w http.ResponseWriter, r *http.Request) {
	var pid int32
	pidArg, ok := gimlet.GetVars(r)["pid"]
	if ok {
		grip.Debugf("found pid '%s', converting argument", pidArg)
		p, err := strconv.Atoi(pidArg)
		if err != nil {
			gimlet.WriteErrorJSON(w, &statsErrorResponse{
				Error: fmt.Sprintf("could not convert '%s' to int32", pidArg),
			})
			return
		}

		pid = int32(p)
	} else if pid <= 0 {
		// if no pid is specified (which can happen as this
		// handler is used for a route without a pid), we
		// should just inspect the root pid of the
		// system. Also Pid 0 isn't a thing.
		pid = 1
	}

	out := message.CollectProcessInfoWithChildren(pid)
	if len(out) == 0 {
		gimlet.WriteErrorJSON(w, &statsErrorResponse{Pid: int(pid),
			Error: "pid not identified"})
		return
	}

	for _, info := range out {
		if !info.Loggable() {
			resp := &statsErrorResponse{Error: strings.Join(info.(*message.ProcessInfo).Errors, "; ")}
			gimlet.WriteInternalErrorJSON(w, resp)
			return
		}
	}

	gimlet.WriteJSON(w, out)
}
