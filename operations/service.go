package operations

import (
	"github.com/mongodb/amboy/rest"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type GreenbayService struct {
	service *rest.Service
}

func NewService(port int) (*GreenbayService, error) {
	s := &GreenbayService{
		// this operation loads all job instance names.
		service: rest.NewService(),
	}

	if err := s.service.App().SetPort(port); err != nil {
		return nil, errors.Wrap(err, "problem constructing greenbay service")
	}

	return s, nil
}

func (s *GreenbayService) Open(ctx context.Context, info rest.ServiceInfo) error {
	// // TODO: add routes to the app here.
	// app := s.service.App()
	// app.AddRoute("/check/suite/{name}").Version(1).Post().Handler()

	if err := s.service.OpenInfo(ctx, info); err != nil {
		return errors.Wrap(err, "problem opening queue")
	}

	return nil
}

func (s *GreenbayService) Close() {
	s.service.Close()
}

func (s *GreenbayService) Run() {
	s.service.Run()
}
