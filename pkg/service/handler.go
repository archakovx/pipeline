//Package service http handler
//CODE GENERATED AUTOMATICALLY
//THIS FILE COULD BE EDITED BY HANDS
package service

import (
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

// HandlerSettings ...
type HandlerSettings struct {
	Path    string
	Method  string
	Handler fasthttp.RequestHandler
}

// MakeFastHTTPRouter ...
func MakeFastHTTPRouter(handlerSettings []*HandlerSettings) *fasthttprouter.Router {
	router := fasthttprouter.New()

	for _, settings := range handlerSettings {
		router.Handle(settings.Method, settings.Path, settings.Handler)
	}

	return router
}
