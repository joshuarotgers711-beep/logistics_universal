package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"tracking-svc/internal/db"
	"tracking-svc/internal/httpserver"
	"tracking-svc/internal/repo"

	pyroscope "github.com/grafana/pyroscope-go"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

func maybeStartPprof() {
	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			log.Println("pprof enabled on :6060")
			_ = http.ListenAndServe(":6060", nil)
		}()

	}

}

func maybeStartPyroscope(service string) {
	if os.Getenv("ENABLE_PYROSCOPE") != "true" {
		return
	}
	addr := os.Getenv("PYROSCOPE_SERVER")
	if addr == "" {
		addr = "http://pyroscope:4040"
	}
	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: service,
		ServerAddress:   addr,
		Tags:            map[string]string{"service": service},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
		},
	})
	if err != nil {
		log.Printf("pyroscope start: %v", err)
	}
}

func main() {
	ctx := context.Background()

	// Init OpenTelemetry tracer
	shutdown, err := initTracer(ctx)
	if err != nil {
		log.Printf("otel init failed: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()

	maybeStartPprof()

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "tracking-svc"
	}
	maybeStartPyroscope(serviceName)

	store, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.Migrate(ctx, store); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}
	r := repo.New(store)
	srv := httpserver.NewWithRepo(r)
	wrapped := otelhttp.NewHandler(srv, "tracking-svc")
	log.Printf("tracking-svc listening on %s", port)
	if err := http.ListenAndServe(":"+port, wrapped); err != nil {
		log.Fatal(err)
	}
}

func initTracer(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://otel-collector:4317"
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint[7:]), // strip http://
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock(), grpc.WithTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}
	service := os.Getenv("OTEL_SERVICE_NAME")
	if service == "" {
		service = "tracking-svc"
	}
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(service),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}
