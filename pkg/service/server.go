//Package service http server
//CODE GENERATED AUTOMATICALLY
//THIS FILE COULD BE EDITED BY HANDS
package service

import (
	"context"
	"github.com/valyala/fasthttp"
	"time"
)

type executeServer struct {
	transport      ExecuteTransport
	service        Service
	errorProcessor ErrorProcessor
}

// ServeHTTP implements http.Handler.
func (s *executeServer) ServeHTTP(ctx *fasthttp.RequestCtx) {
	request, err := s.transport.DecodeRequest(ctx, &ctx.Request)
	if err != nil {
		s.errorProcessor.Encode(ctx, &ctx.Response, err)
		return
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	response, err := s.service.Execute(timeoutCtx, &request)
	if err != nil {
		s.errorProcessor.Encode(ctx, &ctx.Response, err)
		return
	}

	if err := s.transport.EncodeResponse(ctx, &ctx.Response, &response); err != nil {
		s.errorProcessor.Encode(ctx, &ctx.Response, err)
		return
	}
}

// NewExecuteServer the server creator
func NewExecuteServer(transport ExecuteTransport, service Service, errorProcessor ErrorProcessor) fasthttp.RequestHandler {
	ls := executeServer{
		transport:      transport,
		service:        service,
		errorProcessor: errorProcessor,
	}
	return ls.ServeHTTP
}
