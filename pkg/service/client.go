//Package service http client
//CODE GENERATED AUTOMATICALLY
//THIS FILE COULD BE EDITED BY HANDS
package service

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/valyala/fasthttp"

	"github.com/pipeline/pkg/models"
)

type client struct {
	cli *fasthttp.HostClient

	transportExecute ExecuteClientTransport
}

// Execute ...
func (s *client) Execute(ctx context.Context, request *models.ExecuteRequest) (response models.ExecuteResponse, err error) {
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	if err = s.transportExecute.EncodeRequest(ctx, req, request); err != nil {
		return
	}
	err = s.cli.Do(req, res)
	if err != nil {
		return
	}
	return s.transportExecute.DecodeResponse(ctx, res)
}

// NewClient the client creator
func NewClient(
	cli *fasthttp.HostClient,

	transportExecute ExecuteClientTransport,
) Service {
	return &client{
		cli: cli,

		transportExecute: transportExecute,
	}
}

// ExecuteClientTransport transport interface
type ExecuteClientTransport interface {
	EncodeRequest(ctx context.Context, r *fasthttp.Request, request *models.ExecuteRequest) (err error)
	DecodeResponse(ctx context.Context, r *fasthttp.Response) (response models.ExecuteResponse, err error)
}

type executeClientTransport struct {
	errorProcessor ErrorProcessor
	errorCreator   ErrorCreator
	pathTemplate   string
	method         string
}

// EncodeRequest method for encoding requests on client side
func (t *executeClientTransport) EncodeRequest(ctx context.Context, r *fasthttp.Request, request *models.ExecuteRequest) (err error) {
	r.Header.SetMethod(t.method)
	r.SetRequestURI(t.pathTemplate)
	r.Header.Set("Content-Type", "application/json")
	r.SetBodyStreamWriter(func(w *bufio.Writer) {
		if err = json.NewEncoder(w).Encode(request); err != nil {
			return
		}
	})
	return
}

// DecodeResponse method for decoding response on client side
func (t *executeClientTransport) DecodeResponse(ctx context.Context, r *fasthttp.Response) (response models.ExecuteResponse, err error) {
	if r.StatusCode() != http.StatusOK {
		err = t.errorProcessor.Decode(r)
		return
	}
	if err = response.UnmarshalJSON(r.Body()); err != nil {
		log.Printf("error while decoding response: %v, value: %v", err, response)
	}
	return
}

// NewExecuteClientTransport the transport creator for http requests
func NewExecuteClientTransport(
	errorProcessor ErrorProcessor,
	errorCreator ErrorCreator,
	pathTemplate string,
	method string,
) ExecuteClientTransport {
	return &executeClientTransport{
		errorProcessor: errorProcessor,
		errorCreator:   errorCreator,
		pathTemplate:   pathTemplate,
		method:         method,
	}
}
