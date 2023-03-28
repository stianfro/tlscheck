package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func main() {
	// Initialize OpenTelemetry with Prometheus exporter
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{})
	if err != nil {
		log.Fatalf("Failed to create Prometheus exporter: %v", err)
	}
	http.HandleFunc("/", exporter.ServeHTTP)
	go func() {
		_ = http.ListenAndServe(":2112", nil)
	}()

	// Set global OpenTelemetry options
	res, _ := resource.New(context.Background(), resource.WithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("tlscheck"),
	))
	otel.SetResource(res)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Initialize Kubernetes client
	clientset, err := initKubeClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	fmt.Printf("%-30s %5s\n", "CERTIFICATE_NAME", "REMAINING_LIFETIME (days)")
	fmt.Println("---------------------------------------------------")

	// Check TLS certificates and export metrics
	checkTLSCertificates(context.Background(), clientset)
}
