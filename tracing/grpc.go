package tracing

import (
	"context"

	oldCtx "golang.org/x/net/context"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcOpentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/sirupsen/logrus"
	"github.com/windmilleng/mish/logging"
	"google.golang.org/grpc"
)

func addTraceIDFields(ctx context.Context) {
	span := SystemSpanFromContext(ctx)
	if span == nil {
		return
	}

	traceID := TraceIDFromSpan(span)
	if traceID == "" {
		return
	}

	logging.AddFields(ctx, logrus.Fields{
		"traceId": traceID,
	})
}

func NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := SystemTracer()
	options := grpcOpentracing.WithTracer(tracer)

	return middleware.ChainUnaryServer(
		grpcOpentracing.UnaryServerInterceptor(options),
		func(ctx oldCtx.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			addTraceIDFields(ctx)
			return handler(ctx, req)
		})
}

func NewStreamServerInterceptor() grpc.StreamServerInterceptor {
	tracer := SystemTracer()
	options := grpcOpentracing.WithTracer(tracer)

	return middleware.ChainStreamServer(
		grpcOpentracing.StreamServerInterceptor(options),
		func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			addTraceIDFields(stream.Context())
			return handler(srv, stream)
		})
}

func NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	tracer := SystemTracer()
	options := grpcOpentracing.WithTracer(tracer)
	return grpcOpentracing.UnaryClientInterceptor(options)
}

func NewStreamClientInterceptor() grpc.StreamClientInterceptor {
	tracer := SystemTracer()
	options := grpcOpentracing.WithTracer(tracer)
	return grpcOpentracing.StreamClientInterceptor(options)
}
