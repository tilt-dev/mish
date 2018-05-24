package tracing

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

// Inspired by
// https://github.com/opentracing/opentracing-go/blob/master/gocontext.go
// but with our two-tracer division of labor.
//
// For system spans we defer to the default implementation with a global
// tracer so that we get gRPC propagation for free.

type contextKey struct{}

var userSpanKey = contextKey{}

func ContextWithUserSpan(ctx context.Context, span opentracing.Span) context.Context {
	return context.WithValue(ctx, userSpanKey, span)
}

func ContextWithSystemSpan(ctx context.Context, span opentracing.Span) context.Context {
	return opentracing.ContextWithSpan(ctx, span)
}

func UserSpanFromContext(ctx context.Context) opentracing.Span {
	return spanFromContext(ctx, userSpanKey)
}

func SystemSpanFromContext(ctx context.Context) opentracing.Span {
	return opentracing.SpanFromContext(ctx)
}

func spanFromContext(ctx context.Context, key contextKey) opentracing.Span {
	val := ctx.Value(key)
	if sp, ok := val.(opentracing.Span); ok {
		return sp
	}
	return nil
}

func StartUserSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return startSpanFromContextWithTracer(ctx, userSpanKey, UserTracer(), operationName, opts...)
}

func StartSystemSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operationName, opts...)
}

func startSpanFromContextWithTracer(ctx context.Context, key contextKey, tracer opentracing.Tracer, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	var span opentracing.Span
	parentSpan := spanFromContext(ctx, key)
	if parentSpan != nil {
		opts = append(opts, opentracing.ChildOf(parentSpan.Context()))
		span = tracer.StartSpan(operationName, opts...)
	} else {
		span = tracer.StartSpan(operationName, opts...)
	}
	return span, context.WithValue(ctx, key, span)
}
