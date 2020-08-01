package service

import (
	"context"
	"errors"
	"github.com/pipeline/pkg/models"
)

type Service interface {
	Execute(ctx context.Context, request *models.ExecuteRequest) (response models.ExecuteResponse, err error)
}

type service struct {
	Jobs []models.Job
}

func (s *service) Execute(ctx context.Context, request *models.ExecuteRequest) (response models.ExecuteResponse, err error) {
	input := make(chan interface{})
	for i := range s.Jobs {
		output := make(chan interface{})
		go func(i int, input, output chan interface{}) {
			s.Jobs[i](input, output)
			close(output)
		}(i, input, output)

		input = output
	}

	for {
		select {
		case <-ctx.Done():
			err = errors.New("pipeline took too long")
			return
		case <-input:
			return
		}
	}
}

func NewService(jobs []models.Job) Service {
	return &service{Jobs: jobs}
}
