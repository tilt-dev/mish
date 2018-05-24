package tracing

import (
	"fmt"
	"net/url"

	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/server/wmservice"
)

// Only trace 1/N system rpcs
const defaultSystemSampleRate = 1000

var systemTracer opentracing.Tracer = &opentracing.NoopTracer{}

var userTracer opentracing.Tracer = &opentracing.NoopTracer{}

var viewURL url.URL = url.URL{}

/**
 * Traces Windmill system RPCs.
 *
 * Automatically propagated across gRPC.
 *
 * Intended for use by Windmill developers for tracing slow app performance.
 */
func SystemTracer() opentracing.Tracer {
	return systemTracer
}

/**
 * Traces Windmill Compositions and Runs.
 *
 * Intended for use by Windmill users for tracing which of their tasks are slow.
 *
 * We may also use this in the future to collect aggregate data about
 * composition pass/fail rates and run times, and use that data to
 * make prioritization decisions.
 */
func UserTracer() opentracing.Tracer {
	return userTracer
}

type TraceID string

/**
 * A unique identifier for the entire trace (across all spans)
 */
func TraceIDFromSpan(span opentracing.Span) TraceID {
	ctx := span.Context()
	zipkinCtx, ok := ctx.(zipkin.SpanContext)
	if !ok {
		return ""
	}
	traceID := zipkinCtx.TraceID
	if traceID.Empty() {
		return ""
	}
	return TraceID(traceID.ToHex())
}

type ZipkinOptions struct {
	viewURL          url.URL
	collector        zipkin.Collector
	systemSampleRate int
	err              error
}

func PublicTracerURL(tag string) url.URL {
	str := "http://trace-collector.windmill.engineering/"
	if tag == "prod" {
		str = "http://tracer.windmill.build/"
	}

	uri, err := url.Parse(str)
	if err != nil {
		panic(err)
	}
	return *uri
}

func DefaultZipkinOptions() ZipkinOptions {
	return ZipkinOptions{}.WithSystemSampleRate(defaultSystemSampleRate)
}

func (z ZipkinOptions) WithTag(tag string) ZipkinOptions {
	z.viewURL = PublicTracerURL(tag)
	return z
}

// Measure 1/N of system traces.
// For easy use with flags, if this sets the rate to 0, we keep the default rate.
func (z ZipkinOptions) WithSystemSampleRate(sampleRate int) ZipkinOptions {
	if sampleRate != 0 {
		z.systemSampleRate = sampleRate
	}
	return z
}

func (z ZipkinOptions) WithCollectorAddr(addr string) ZipkinOptions {
	collector, err := zipkin.NewHTTPCollector(fmt.Sprintf("http://%s/api/v1/spans", addr))
	if err != nil {
		z.err = errors.Propagatef(err, "Error creating Zipkin collector")
	} else {
		z.collector = collector
	}
	return z
}

/**
 * The Tracing collectors are administered much differently than our other services.
 *
 * In a future production environment, we would run two sets of collectors:
 * one for system traces (Windmill system perf) and one for user traces (Jobs run on Windmill).
 * User traces would be tiered like any other storage server and persisted forever.
 * System traces would make more sense aggregated across tiers, tagged, and expired every N days
 * (i.e., our dev machines and our prod machines could send their traces to the same collector).
 *
 * For the prototype, we run one persistent collector for all traces (user and system) for ease
 * of use, and reboot it rarely.
 */
func SendTracesToZipkin(options ZipkinOptions) error {
	if options.err != nil {
		return options.err
	}

	var err error

	// Set the global tracer url
	viewURL = options.viewURL

	serviceName, err := wmservice.ServiceName()
	if err != nil {
		return err
	}

	serviceTag, err := wmservice.ServiceTag()
	if err != nil {
		return err
	}

	userService := fmt.Sprintf("windmill:%s", serviceTag)
	userRecorder := zipkin.NewRecorder(options.collector, false, "0.0.0.0:0", userService)
	userTracer, err = zipkin.NewTracer(userRecorder)
	if err != nil {
		return err
	}

	service := fmt.Sprintf("%s:%s", serviceName, serviceTag)
	systemRecorder := zipkin.NewRecorder(options.collector, false, "0.0.0.0:0", service)
	sampler := zipkin.ModuloSampler(uint64(options.systemSampleRate))
	systemTracer, err = zipkin.NewTracer(systemRecorder, zipkin.WithSampler(sampler))
	if err != nil {
		return err
	}

	opentracing.SetGlobalTracer(systemTracer)

	return nil
}

// Create in-memory tracers. We use tracers to track all timing data,
// so this is useful when we need to test whether timing data is getting
// propagated properly throughout the system.
func SetupTracersForTesting() error {
	userRecorder := zipkin.NewInMemoryRecorder()
	systemRecorder := zipkin.NewInMemoryRecorder()

	var err error
	userTracer, err = zipkin.NewTracer(userRecorder)
	if err != nil {
		return err
	}

	systemTracer, err = zipkin.NewTracer(systemRecorder)
	if err != nil {
		return err
	}

	return nil
}

func ResetTracersForTesting() {
	systemTracer = &opentracing.NoopTracer{}
	userTracer = &opentracing.NoopTracer{}
	viewURL = url.URL{}
}

func TraceIDToURL(id TraceID) url.URL {
	if viewURL == (url.URL{}) || id == "" {
		return url.URL{}
	}

	traceURL := viewURL
	traceURL.Path = fmt.Sprintf("/zipkin/traces/%s", id)
	return traceURL
}
