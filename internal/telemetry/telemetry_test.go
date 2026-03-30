package telemetry

import (
	"context"
	"os"
	"testing"
)

func TestInit_NoEndpoint(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown, err := Init(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}
}

func TestInit_WithEndpointInsecure(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	defer func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
	}()

	// otlptracegrpc dial é lazy — Init não falha mesmo sem servidor real
	shutdown, err := Init(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("expected no error (lazy dial), got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	_ = shutdown(context.Background())
}

func TestInit_WithEndpointSecure(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
	defer func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
	}()

	shutdown, err := Init(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("expected no error (lazy dial), got %v", err)
	}
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}
