
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Initialize the OTLP/HTTP exporter.
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize exporter: %v", err)
	}

	// Configure the tracing provider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				"",
				attribute.String("service.name", "hello-api"),
			),
		),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Flush pending spans before exiting.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf("Tracing shutdown error: %v", err)
		}
	}()

	mux := http.NewServeMux()

	// Kubernetes health endpoints.
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", healthHandler)

	// Instrument application HTTP requests.
	mux.Handle(
		"/",
		otelhttp.NewHandler(
			http.HandlerFunc(getHello),
			"hello-api.request",
		),
	)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("hello-api listening on port 8080")

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK\n")
}

func getHello(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("hello-api").Start(
		r.Context(),
		"generate-greeting",
	)
	defer span.End()

	name := r.URL.Query().Get("name")

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "Hello "+name+"\n")

	log.Printf(
		"greeting served trace_id=%s",
		span.SpanContext().TraceID().String(),
	)

	_ = ctx
}
