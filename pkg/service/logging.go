//Package service logging wrapper
//CODE GENERATED AUTOMATICALLY
//THIS FILE COULD BE EDITED BY HANDS
package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/pipeline/pkg/models"
)

// loggingMiddleware wraps Service and logs request information to the provided logger
type loggingMiddleware struct {
	logger log.Logger
	svc    Service
}

func (s *loggingMiddleware) Execute(ctx context.Context, request *models.ExecuteRequest) (response models.ExecuteResponse, err error) {
	defer func(begin time.Time) {
		_ = s.wrap(err).Log(
			"method", "Execute",
			"request", request,
			"err", err,
			"elapsed", time.Since(begin),
		)
	}(time.Now())
	return s.svc.Execute(ctx, request)
}

func (s *loggingMiddleware) wrap(err error) log.Logger {
	lvl := level.Debug
	if err != nil {
		lvl = level.Error
	}
	return lvl(s.logger)
}

// NewLoggingMiddleware ...
func NewLoggingMiddleware(logger log.Logger, svc Service) Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}
