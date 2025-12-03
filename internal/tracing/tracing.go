// Package tracing provides OpenTelemetry tracing for AGIS.
package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration.
type Config struct {
	Enabled     bool
	ServiceName string
	Environment string
	Endpoint    string  // OTLP endpoint (e.g., "localhost:4317")
	SampleRate  float64 // 0.0 to 1.0
	Insecure    bool    // Use insecure connection
}

// DefaultConfig returns sensible defaults for development.
func DefaultConfig() Config {
	return Config{
		Enabled:     false,
		ServiceName: "agis-bot",
		Environment: "development",
		Endpoint:    "localhost:4317",
		SampleRate:  1.0,
		Insecure:    true,
	}
}

// Provider wraps the tracer provider with convenience methods.
type Provider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// Init initializes the tracing provider.
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		// Return a no-op provider
		noop := trace.NewNoopTracerProvider()
		otel.SetTracerProvider(noop)
		return &Provider{
			provider: nil,
			tracer:   noop.Tracer(cfg.ServiceName),
		}, nil
	}

	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Create resource with service info
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
			attribute.String("service.version", "1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create sampler
	var sampler sdktrace.Sampler
	switch {
	case cfg.SampleRate <= 0:
		sampler = sdktrace.NeverSample()
	case cfg.SampleRate >= 1:
		sampler = sdktrace.AlwaysSample()
	default:
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		provider: tp,
		tracer:   tp.Tracer(cfg.ServiceName),
	}, nil
}

// Shutdown gracefully shuts down the tracer provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.provider != nil {
		return p.provider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the underlying tracer.
func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

// StartSpan starts a new span with the given name and attributes.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer("agis-bot")
	ctx, span := tracer.Start(ctx, name)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// SetStatus sets the status on the current span.
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// TraceID returns the trace ID from the context.
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().TraceID().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// SpanID returns the span ID from the context.
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().SpanID().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// DiscordAttributes returns common Discord-related span attributes.
func DiscordAttributes(guildID, channelID, userID string) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if guildID != "" {
		attrs = append(attrs, attribute.String("discord.guild_id", guildID))
	}
	if channelID != "" {
		attrs = append(attrs, attribute.String("discord.channel_id", channelID))
	}
	if userID != "" {
		attrs = append(attrs, attribute.String("discord.user_id", userID))
	}
	return attrs
}

// statusWriter wraps http.ResponseWriter to capture status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns HTTP middleware that adds tracing to requests.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := StartSpan(r.Context(), r.URL.Path,
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
		)
		defer span.End()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r.WithContext(ctx))

		span.SetAttributes(attribute.Int("http.status_code", sw.status))
		if sw.status >= 400 {
			span.SetStatus(codes.Error, http.StatusText(sw.status))
		}
	})
}

// Timed executes a function within a traced span.
func Timed(ctx context.Context, name string, fn func(ctx context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
	}
	return err
}
