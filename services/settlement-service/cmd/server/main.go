package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/movra/settlement-service/internal/config"
	settlementgrpc "github.com/movra/settlement-service/internal/grpc"
	"github.com/movra/settlement-service/internal/kafka"
	"github.com/movra/settlement-service/internal/provider"
	"github.com/movra/settlement-service/internal/repository"
	"github.com/movra/settlement-service/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load config
	cfg := config.Load()

	// Setup Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis not available, service will work without persistence", zap.Error(err))
	}

	// Create repository
	repo := repository.NewRedisRepository(redisClient)

	// Create provider
	var payoutProvider provider.PayoutProvider
	switch cfg.ProviderType {
	case "simulated":
		payoutProvider = provider.NewSimulatedProvider(cfg.ProviderFailureRate, cfg.ProviderProcessingTime)
	default:
		payoutProvider = provider.NewSimulatedProvider(10, 2*time.Second)
	}

	// Create service
	payoutService := service.NewPayoutService(repo, payoutProvider, logger, cfg.MaxRetries)

	// Setup Gin router for HTTP
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "settlement-service",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "settlement-service",
		})
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	settlementServer := settlementgrpc.NewSettlementServer(payoutService, logger)
	settlementgrpc.RegisterSettlementServiceServer(grpcServer, settlementServer)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	// Create Kafka consumer
	kafkaConsumer := kafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaTopicFunded,
		cfg.KafkaConsumerGroup,
		payoutService,
		logger,
	)

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", zap.String("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
		if err != nil {
			logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}
		logger.Info("Starting gRPC server", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// Start Kafka consumer
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	go func() {
		if err := kafkaConsumer.Start(consumerCtx); err != nil {
			logger.Error("Kafka consumer error", zap.Error(err))
		}
	}()

	logger.Info("Settlement Service started",
		zap.String("httpPort", cfg.HTTPPort),
		zap.String("grpcPort", cfg.GRPCPort),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")

	// Graceful shutdown
	cancelConsumer()
	kafkaConsumer.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", zap.Error(err))
	}

	logger.Info("Settlement Service stopped")
}
