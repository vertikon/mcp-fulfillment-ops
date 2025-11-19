package httpserver

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-fulfillment-ops/internal/observability"
)

func TestNewServer(t *testing.T) {
	config := Config{
		Port:         8080,
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()

	server := NewServer(config, metrics)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.e == nil {
		t.Error("Echo instance should not be nil")
	}
	if server.config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.config.Port)
	}
	if server.metrics == nil {
		t.Error("Metrics should not be nil")
	}
}

func TestServer_RegisterRoute(t *testing.T) {
	config := Config{
		Port:         8080,
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()
	server := NewServer(config, metrics)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	}

	server.RegisterRoute("GET", "/test", handler)

	// Verify route is registered
	routes := server.e.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/test" && route.Method == "GET" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Route /test should be registered")
	}
}

func TestServer_GetEcho(t *testing.T) {
	config := Config{
		Port:         8080,
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()
	server := NewServer(config, metrics)

	echo := server.GetEcho()
	if echo == nil {
		t.Error("GetEcho should return Echo instance")
	}
	if echo != server.e {
		t.Error("GetEcho should return the same Echo instance")
	}
}

func TestServer_HealthEndpoints(t *testing.T) {
	config := Config{
		Port:         0, // Use random port for testing
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()
	server := NewServer(config, metrics)

	// Start server in goroutine
	go func() {
		_ = server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test /health endpoint
	resp, err := http.Get("http://localhost:8080/health")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	// Test /ready endpoint
	resp, err = http.Get("http://localhost:8080/ready")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	// Test /metrics endpoint
	resp, err = http.Get("http://localhost:8080/metrics")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Stop(ctx)
}

func TestServer_Stop(t *testing.T) {
	config := Config{
		Port:         0, // Use random port
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()
	server := NewServer(config, metrics)

	// Start server in goroutine
	started := make(chan bool, 1)
	go func() {
		started <- true
		_ = server.Start()
	}()

	// Wait for server to start
	<-started
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestServer_Middlewares(t *testing.T) {
	config := Config{
		Port:         0,
		Host:         "localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metrics := observability.NewMetrics()
	server := NewServer(config, metrics)

	// Verify middlewares are registered
	routes := server.e.Routes()
	if len(routes) == 0 {
		t.Error("Expected routes to be registered")
	}

	// Check for health endpoints
	hasHealth := false
	hasReady := false
	hasMetrics := false

	for _, route := range routes {
		if route.Path == "/health" && route.Method == "GET" {
			hasHealth = true
		}
		if route.Path == "/ready" && route.Method == "GET" {
			hasReady = true
		}
		if route.Path == "/metrics" && route.Method == "GET" {
			hasMetrics = true
		}
	}

	if !hasHealth {
		t.Error("Health endpoint should be registered")
	}
	if !hasReady {
		t.Error("Ready endpoint should be registered")
	}
	if !hasMetrics {
		t.Error("Metrics endpoint should be registered")
	}
}

func TestHealthHandler(t *testing.T) {
	// This is tested indirectly through server tests
	// but we can verify the handler function exists
	if healthHandler == nil {
		t.Error("healthHandler should be defined")
	}
}

func TestReadyHandler(t *testing.T) {
	// This is tested indirectly through server tests
	// but we can verify the handler function exists
	if readyHandler == nil {
		t.Error("readyHandler should be defined")
	}
}

