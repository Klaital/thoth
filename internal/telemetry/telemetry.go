// Package telemetry wires up OpenTelemetry traces, metrics and logs.
//
// Everything is configured through the standard OTEL_* environment variables
// (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_HEADERS, OTEL_SERVICE_NAME,
// OTEL_RESOURCE_ATTRIBUTES, ...), so pointing at a different backend is a
// config change. When no OTLP endpoint is configured, Setup leaves the global
// no-op providers in place and only installs the stdout logger, so tests and
// local runs behave exactly as before.
package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// metricInterval is how often metrics are pushed. 60s keeps Grafana Cloud's
// free-tier sample budget comfortable.
const metricInterval = 60 * time.Second

// Enabled reports whether an OTLP endpoint is configured and the SDK is not
// explicitly disabled.
func Enabled() bool {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return false
	}
	for _, k := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// Setup installs global tracer, meter and logger providers exporting over
// OTLP/HTTP, and makes slog (and therefore the standard log package) write to
// both stdout and OTLP. serviceName is used unless OTEL_SERVICE_NAME is set.
//
// The returned shutdown function flushes and stops all providers; call it
// before the process exits.
func Setup(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error) {
	stdout := slog.NewTextHandler(os.Stdout, nil)

	if !Enabled() {
		slog.SetDefault(slog.New(stdout))
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithFromEnv(), // OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES override
		resource.WithHost(),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
	)
	if err != nil {
		return nil, err
	}

	var shutdowns []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var errs []error
		// Reverse order: logs last so shutdown problems still get logged.
		for i := len(shutdowns) - 1; i >= 0; i-- {
			errs = append(errs, shutdowns[i](ctx))
		}
		return errors.Join(errs...)
	}
	fail := func(e error) (func(context.Context) error, error) {
		return nil, errors.Join(e, shutdown(ctx))
	}

	// Logs first, so everything after this point is captured.
	logExp, err := otlploghttp.New(ctx)
	if err != nil {
		return fail(err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
	)
	global.SetLoggerProvider(lp)
	shutdowns = append(shutdowns, lp.Shutdown)
	slog.SetDefault(slog.New(slog.NewMultiHandler(
		stdout,
		otelslog.NewHandler("github.com/klaital/thoth", otelslog.WithLoggerProvider(lp)),
	)))

	traceExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return fail(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	shutdowns = append(shutdowns, tp.Shutdown)

	metricExp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return fail(err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(metricInterval))),
	)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	if err := runtime.Start(); err != nil {
		return fail(err)
	}

	// Export failures go to stdout only: routing them through the OTLP log
	// pipeline would feed failures back into the exporter that is failing.
	stdoutLog := slog.New(stdout)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		stdoutLog.Warn("otel error", "err", err)
	}))

	slog.Info("telemetry enabled", "service", serviceName)
	return shutdown, nil
}
