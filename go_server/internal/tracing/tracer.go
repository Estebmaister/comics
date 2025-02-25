package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	shutdownTimeout = 5 * time.Second
	errMessage      = "error occurred"
)

// TracerConfig holds the tracer configuration
type TracerConfig struct {
	Environment string `mapstructure:"ENVIRONMENT" default:"development"`
	Endpoint    string `mapstructure:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"`
	Secure      bool   `mapstructure:"OTEL_EXPORTER_OTLP_SECURE" default:"false"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME" default:"comics-service"`
	Sampler     int    `mapstructure:"OTEL_TRACES_SAMPLER_PERCENTAGE" default:"100"`
}

// Tracer holds the tracer provider and tracer created
type Tracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

// NewTracer creates a new OpenTelemetry tracer
func NewTracer(ctx context.Context, cfg TracerConfig, namespace string) (*Tracer, error) {
	// Set OpenTelemetry to use Zerolog via logr adapter
	// otel.SetLogger(zerologr.New(&log.Logger))

	// Set up propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Initialize OTLP client
	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpointURL(cfg.Endpoint),
	}
	if !cfg.Secure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}
	client := otlptracegrpc.NewClient(clientOpts...)

	// Create OTLP exporter
	var exporter tracesdk.SpanExporter
	var err error
	if cfg.Environment == "developmen" {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else {
		exporter, err = otlptrace.New(ctx, client)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %v", err)
	}

	// Create resource
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		semconv.ServiceNamespaceKey.String(namespace),
	)

	// Create tracer provider
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			tracesdk.WithBatchTimeout(time.Second)),
		tracesdk.WithResource(resources),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(float64(cfg.Sampler)/100)),
	)

	// Set the provider as the global tracer provider
	otel.SetTracerProvider(provider)

	return &Tracer{
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
	}, nil
}

// Shutdown stops the remote tracer provider if it was initialized
func (t *Tracer) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout) // Timeout to prevent blocking shutdown
	defer cancel()

	if provider, ok := t.provider.(*tracesdk.TracerProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("tracer shutdown failed: %w", err)
		}
	}
	return nil
}

// StartSpan starts a new span, it's required to end it
func (t *Tracer) StartSpan(ctx context.Context, operation string) (context.Context, *span) {
	ctx, spanTrace := t.tracer.Start(ctx, operation)
	return ctx, &span{span: spanTrace}
}

// FromContext returns the span from the context if it exists.
// If it doesn't, it returns an implementation of a Span that performs no operations.
func FromContext(ctx context.Context) *span {
	return &span{span: trace.SpanFromContext(ctx)}
}

type span struct {
	span trace.Span
}

// End ends the span
func (s *span) End() {
	s.span.End()
}

// SetError sets the span status to Error and records the error
func (s *span) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// SetOk sets the span status to Ok
func (s *span) SetOk() {
	s.span.SetStatus(codes.Ok, "")
}

// SetTag sets a tag on the span
func (s *span) SetTag(key string, value interface{}) {
	switch v := value.(type) {
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprint(v)))
	}
}
