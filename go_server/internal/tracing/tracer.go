package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

type span struct {
	span trace.Span
}

func (s *span) End() {
	s.span.End()
}

func (s *span) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

func (s *span) SetTag(key string, value interface{}) {
	s.span.SetAttributes(attribute.String(key, fmt.Sprint(value)))
}

func NewTracer(serviceName string, endpoint string) (*Tracer, error) {
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %v", err)
	}

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		attribute.String("environment", "production"),
	)

	provider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resources),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	otel.SetTracerProvider(provider)

	return &Tracer{
		provider: provider,
		tracer:   provider.Tracer(serviceName),
	}, nil
}

func (t *Tracer) StartSpan(ctx context.Context, operation string) (context.Context, *span) {
	ctx, spanTrace := t.tracer.Start(ctx, operation)
	return ctx, &span{span: spanTrace}
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if provider, ok := t.provider.(*tracesdk.TracerProvider); ok {
		return provider.Shutdown(ctx)
	}
	return nil
}
