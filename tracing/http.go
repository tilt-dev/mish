// HTTP middleware for tracing
//
// Weirdly, I was not able to find an opensource implementation for this, possibly
// because it depends more on the JS details than the Go side.

package tracing

import (
	"net/http"

	"github.com/gorilla/mux"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/windmilleng/mish/logging"
)

func TracingHandler(handler http.Handler, router *mux.Router) http.Handler {
	return tracingHandler{
		handler: handler,
		router:  router,
		tracer:  SystemTracer(),
	}
}

type tracingHandler struct {
	handler http.Handler
	router  *mux.Router
	tracer  opentracing.Tracer
}

func (h tracingHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var match mux.RouteMatch
	matched := h.router.Match(req, &match)
	opName := ""
	if matched && match.Route != nil {
		opName = match.Route.GetName()
	}

	if opName == "" {
		opName = "http"
	}

	span := h.tracer.StartSpan(opName, opentracing.Tags{
		"path":                req.URL.Path,
		string(ext.Component): "http",
	})
	defer span.Finish()

	ctx := ContextWithSystemSpan(req.Context(), span)
	req = req.WithContext(ctx)

	traceID := TraceIDFromSpan(span)
	traceURL := TraceIDToURL(traceID)
	traceURLString := traceURL.String()
	if traceURLString != "" {
		res.Header().Set("x-trace-url", traceURLString)
	}

	logging.AddFields(ctx, logrus.Fields{
		"traceId": traceID,
	})

	h.handler.ServeHTTP(res, req)
}
