package tracing

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	// ServiceName identifies the service
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment (production, staging, development)
	Environment string
	// JaegerEndpoint for exporting traces (e.g., "http://localhost:14268/api/traces")
	JaegerEndpoint string
	// SamplingRate (0.0 to 1.0) - 1.0 means sample all traces
	SamplingRate float64
	// Enabled toggles tracing on/off
	Enabled bool
}

// TracingManager manages OpenTelemetry tracing
type TracingManager struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	config   Config
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config Config) (*TracingManager, error) {
	if !config.Enabled {
		// Return no-op tracer
		return &TracingManager{
			tracer: otel.Tracer(config.ServiceName),
			config: config,
		}, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.JaegerEndpoint),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global provider
	otel.SetTracerProvider(provider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &TracingManager{
		tracer:   provider.Tracer(config.ServiceName),
		provider: provider,
		config:   config,
	}, nil
}

// Shutdown gracefully shuts down the tracing provider
func (t *TracingManager) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Middleware returns a Gin middleware for automatic tracing
func (t *TracingManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !t.config.Enabled {
			c.Next()
			return
		}

		// Extract context from incoming request
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)

		// Start span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := t.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// Set span attributes
		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPRoute(c.FullPath()),
			semconv.HTTPTarget(c.Request.URL.Path),
			semconv.HTTPScheme(c.Request.URL.Scheme),
			semconv.HTTPClientIP(c.ClientIP()),
			semconv.HTTPUserAgent(c.Request.UserAgent()),
		)

		// Add request ID if available
		if requestID := c.GetString("request_id"); requestID != "" {
			span.SetAttributes(attribute.String("request.id", requestID))
		}

		// Store context in Gin context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Set response attributes
		span.SetAttributes(
			semconv.HTTPStatusCode(c.Writer.Status()),
		)

		// Mark span as error if status >= 500
		if c.Writer.Status() >= 500 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// StartSpan starts a new span from the context
func (t *TracingManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !t.config.Enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, opts...)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
}

// TraceID returns the trace ID from context
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().TraceID().String()
}

// SpanID returns the span ID from context
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().SpanID().String()
}

// Helper functions for common operations

// TraceHTTPClient wraps an HTTP client to propagate trace context
func TraceHTTPClient(ctx context.Context, req interface{}) {
	// This would inject trace context into outgoing HTTP requests
	// Implementation depends on HTTP client library used
}

// TraceDBQuery traces a database query
func TraceDBQuery(ctx context.Context, query string, args ...interface{}) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("").Start(ctx, "db.query")
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
	)
	return ctx, span
}

// TraceGRPCCall traces a gRPC call
func TraceGRPCCall(ctx context.Context, method string) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("").Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("rpc.system", "grpc"),
		attribute.String("rpc.method", method),
	)
	return ctx, span
}

// Example usage:
//
// func main() {
//     // Initialize tracing
//     tracingMgr, err := tracing.NewTracingManager(tracing.Config{
//         ServiceName:    "order-service",
//         ServiceVersion: "2.0.0",
//         Environment:    "production",
//         JaegerEndpoint: "http://localhost:14268/api/traces",
//         SamplingRate:   0.1, // Sample 10% of traces
//         Enabled:        true,
//     })
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer tracingMgr.Shutdown(context.Background())
//
//     // Add tracing middleware
//     router := gin.New()
//     router.Use(tracingMgr.Middleware())
//
//     // Use in handlers
//     router.GET("/orders/:id", func(c *gin.Context) {
//         ctx := c.Request.Context()
//
//         // Create custom span for business logic
//         ctx, span := tracingMgr.StartSpan(ctx, "get_order_from_db")
//         defer span.End()
//
//         // Add attributes
//         tracing.SetAttributes(ctx,
//             attribute.String("order.id", c.Param("id")),
//             attribute.String("user.id", c.GetString("user_id")),
//         )
//
//         // Fetch order
//         order, err := orderRepo.GetByID(ctx, c.Param("id"))
//         if err != nil {
//             tracing.RecordError(ctx, err)
//             span.SetAttributes(attribute.Bool("error", true))
//             c.JSON(500, gin.H{"error": err.Error()})
//             return
//         }
//
//         // Add event
//         tracing.AddEvent(ctx, "order_retrieved",
//             attribute.String("order.status", order.Status),
//         )
//
//         c.JSON(200, order)
//     })
// }
