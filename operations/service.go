package operations

import (
	"github.com/mongodb/amboy/rest"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// GreenbayService holds the configuration and operations for running
// a Greenbay service.
type GreenbayService struct {
	service *rest.Service
}

// NewService constructs a GreenbayService, but does not start the
// service. You will need to run Open to start the underlying workers and
// Run to start the HTTP service. You can set the host to the empty
// string, to bind the service to all interfaces.
func NewService(host string, port int) (*GreenbayService, error) {
	s := &GreenbayService{
		// this operation loads all job instance names from
		// greenbay and and constructs the amboy.rest.Service object.
		service: rest.NewService(),
	}

	if err := s.service.App().SetPort(port); err != nil {
		return nil, errors.Wrap(err, "problem constructing greenbay service")
	}

	if err := s.service.App().SetHost(host); err != nil {
		return nil, errors.Wrap(err, "problem constructing greenbay service")
	}

	return s, nil
}

// Open starts the service, using the configuration structure from the
// amboy.rest package to set the queue size, number of workers, and
// timeout when restarting the service.
func (s *GreenbayService) Open(ctx context.Context, info rest.ServiceInfo) error {
	// // TODO: add routes to the app here.
	// app := s.service.App()
	// app.AddRoute("/check/suite/{name}").Version(1).Post().Handler()

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
	s.service.Run()
}
