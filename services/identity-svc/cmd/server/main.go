package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"identity-svc/internal/db"
	"identity-svc/internal/httpserver"
	"identity-svc/internal/repo"

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

func randHex(n int) string { b := make([]byte, n); _, _ = rand.Read(b); return hex.EncodeToString(b) }

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
		serviceName = "identity-svc"
	}
	maybeStartPyroscope(serviceName)

	store, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.Migrate(ctx, store); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	r := repo.New(store)
	// Seed admin user for local dev
	if cnt, _ := r.CountUsers(ctx); cnt == 0 {
		email := os.Getenv("ADMIN_EMAIL")
		if email == "" {
			email = "admin@example.com"
		}
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			if f := os.Getenv("ADMIN_PASSWORD_FILE"); f != "" {
				if b, err := os.ReadFile(f); err == nil {
					password = strings.TrimSpace(string(b))
				}
			}
		}
		if password == "" {
			password = "admin123"
		}
		salt := randHex(8)
		h := repo.HashPassword(salt, password)
		if _, err := r.CreateUser(ctx, email, salt, h, "ADMIN"); err != nil {
			log.Printf("seed admin failed: %v", err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	srv := httpserver.NewWithRepo(r)
	wrapped := otelhttp.NewHandler(srv, "identity-svc")
	log.Printf("identity-svc listening on %s", port)
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
		otlptracegrpc.WithEndpoint(endpoint[7:]),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock(), grpc.WithTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}
	service := os.Getenv("OTEL_SERVICE_NAME")
	if service == "" {
		service = "identity-svc"
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
