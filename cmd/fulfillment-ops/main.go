package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	natsAdapter "github.com/vertikon/mcp-fulfillment-ops/internal/adapters/nats"
	"github.com/vertikon/mcp-fulfillment-ops/internal/adapters/postgres"
	redisAdapter "github.com/vertikon/mcp-fulfillment-ops/internal/adapters/redis"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	httpHandler "github.com/vertikon/mcp-fulfillment-ops/internal/interfaces/http"
)

func main() {
	// Carregar configuração
	dbURL := getEnv("DATABASE_URL", "postgres://user:password@localhost/fulfillment?sslmode=disable")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	coreInventoryURL := getEnv("CORE_INVENTORY_URL", "http://localhost:8081")
	httpPort := getEnv("HTTP_PORT", ":8080")

	// Inicializar logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting mcp-fulfillment-ops server")

	// Conectar ao banco de dados
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verificar conexão com banco
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Criar repositório
	repo := postgres.NewFulfillmentRepository(db)

	// Conectar ao NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()

	// Criar JetStream usando a API correta
	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal("Failed to create JetStream", zap.Error(err))
	}
	logger.Info("NATS JetStream connection established")

	// Conectar ao Redis
	redisClient, err := redisAdapter.NewRedisClient(redisURL)
	if err != nil {
		logger.Warn("Failed to connect to Redis, continuing without cache", zap.Error(err))
		redisClient = nil
	} else {
		logger.Info("Redis connection established")
		defer redisClient.Close()
	}

	// Criar adapters de logger
	natsLogger := natsAdapter.NewZapLoggerAdapter(logger)
	appLogger := app.NewZapLoggerAdapter(logger)

	// Criar adapters
	inventoryClient := natsAdapter.NewInventoryCommandClient(coreInventoryURL, natsLogger)
	eventPublisher := natsAdapter.NewEventPublisher(js, natsLogger)

	// Criar casos de uso
	receiveGoodsUC := app.NewReceiveGoodsUseCase(repo, inventoryClient, eventPublisher, appLogger)
	shipOrderUC := app.NewShipOrderUseCase(repo, inventoryClient, eventPublisher, appLogger)
	registerReturnUC := app.NewRegisterReturnUseCase(repo, inventoryClient, eventPublisher, appLogger)
	completeTransferUC := app.NewCompleteTransferUseCase(repo, inventoryClient, eventPublisher, appLogger)
	openCycleCountUC := app.NewOpenCycleCountUseCase(repo, eventPublisher, appLogger)
	submitCycleCountUC := app.NewSubmitCycleCountUseCase(repo, inventoryClient, eventPublisher, appLogger)

	// Iniciar subscriber NATS para eventos OMS
	subscriber := natsAdapter.NewFulfillmentSubscriber(js, shipOrderUC, natsLogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		logger.Fatal("Failed to start NATS subscriber", zap.Error(err))
	}
	logger.Info("NATS subscriber started")

	// Configurar router HTTP
	router := httpHandler.Router(
		receiveGoodsUC,
		shipOrderUC,
		registerReturnUC,
		completeTransferUC,
		openCycleCountUC,
		submitCycleCountUC,
	)

	// Configurar servidor HTTP
	srv := &http.Server{
		Addr:         httpPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Iniciar servidor em goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	cancel() // Cancela contexto do subscriber

	logger.Info("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
