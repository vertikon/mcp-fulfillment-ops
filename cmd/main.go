// Package main provides the HTTP server entry point for MCP-Hulk
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/vertikon/mcp-fulfillment-ops/internal/core/cache"
	"github.com/vertikon/mcp-fulfillment-ops/internal/core/config"
	"github.com/vertikon/mcp-fulfillment-ops/internal/core/engine"
	"github.com/vertikon/mcp-fulfillment-ops/internal/core/events"
	"github.com/vertikon/mcp-fulfillment-ops/internal/core/scheduler"
	"github.com/vertikon/mcp-fulfillment-ops/internal/observability"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/httpserver"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfgLoader := config.NewLoader()
	cfg, err := cfgLoader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	envMgr := config.NewEnvironmentManager()
	if err := logger.Init(cfg.Logging.Level, envMgr.IsDevelopment()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting MCP-Hulk server")

	// Initialize observability
	var tracerProvider *observability.TracerProvider
	if cfg.Telemetry.Tracing.Enabled {
		tracerProvider, err = observability.InitTracing("mcp-fulfillment-ops", cfg.Telemetry.Tracing.Endpoint)
		if err != nil {
			logger.Error("Failed to initialize tracing", zap.Error(err))
		} else {
			defer tracerProvider.Shutdown(context.Background())
		}
	}

	metrics := observability.NewMetrics()

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATS.URLs[0])
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Fatal("Failed to get JetStream context", zap.Error(err))
	}

	// Initialize scheduler with NATS
	taskScheduler := scheduler.NewScheduler(js)
	if err := taskScheduler.InitializeStreams(context.Background()); err != nil {
		logger.Fatal("Failed to initialize NATS streams", zap.Error(err))
	}

	// Initialize event publisher
	eventPublisher := events.NewEventPublisher(js)

	// Initialize cache (L1 only for now, L2/L3 can be added later)
	cacheInstance := cache.NewMultiLevelCache(cfg.Cache.L1Size, nil, nil)

	// Initialize execution engine
	workers := config.GetEngineWorkers(&cfg.Engine)
	execEngine := engine.NewExecutionEngine(workers, cfg.Engine.QueueSize, cfg.Engine.Timeout)

	// Start execution engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := execEngine.Start(ctx); err != nil {
		logger.Fatal("Failed to start execution engine", zap.Error(err))
	}
	defer execEngine.Stop()

	// Start scheduler tick publisher
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := taskScheduler.PublishTick(ctx); err != nil {
					logger.Error("Failed to publish tick", zap.Error(err))
				}
			}
		}
	}()

	// Initialize HTTP server
	serverConfig := httpserver.Config{
		Port:         cfg.Server.Port,
		Host:         cfg.Server.Host,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	server := httpserver.NewServer(serverConfig, metrics)

	// Start HTTP server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Publish initial health event
	eventPublisher.PublishRuntimeHealth(ctx, true, map[string]interface{}{
		"workers": workers,
		"port":    cfg.Server.Port,
	})

	logger.Info("MCP-Hulk server started",
		zap.Int("port", cfg.Server.Port),
		zap.Int("workers", workers),
	)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

