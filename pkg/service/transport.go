//Package service http transport
//CODE GENERATED AUTOMATICALLY
//THIS FILE COULD BE EDITED BY HANDS
package service

import (
	"context"
	"net/http"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"

	"github.com/pipeline/pkg/models"
)

// ExecuteTransport transport interface
type ExecuteTransport interface {
	DecodeRequest(ctx context.Context, r *fasthttp.Request) (request models.ExecuteRequest, err error)
	EncodeResponse(ctx context.Context, r *fasthttp.Response, response *models.ExecuteResponse) (err error)
}

type executeTransport struct {
	errorCreator ErrorCreator
}

// DecodeRequest method for decoding requests on server side
func (t *executeTransport) DecodeRequest(ctx context.Context, r *fasthttp.Request) (request models.ExecuteRequest, err error) {
	if err = request.UnmarshalJSON(r.Body()); err != nil {
		return models.ExecuteRequest{}, t.errorCreator(
			http.StatusBadRequest,
			"failed to decode JSON request: %v",
			err,
		)
	}
	return
}

// EncodeResponse method for encoding response on server side
func (t *executeTransport) EncodeResponse(ctx context.Context, r *fasthttp.Response, response *models.ExecuteResponse) (err error) {
	r.Header.Set("Content-Type", "application/json")
	if _, err = easyjson.MarshalToWriter(response, r.BodyWriter()); err != nil {
		return t.errorCreator(http.StatusInternalServerError, "failed to encode JSON response: %s", err)
	}
	return
}

// NewExecuteTransport the transport creator for http requests
func NewExecuteTransport(
	errorCreator ErrorCreator,
) ExecuteTransport {
	return &executeTransport{
		errorCreator: errorCreator,
	}
}
