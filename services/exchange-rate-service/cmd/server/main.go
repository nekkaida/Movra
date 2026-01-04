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
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/config"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/handler"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/metrics"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/provider"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/repository"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	grpcserver "github.com/patteeraL/movra/services/exchange-rate-service/internal/grpc"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	logger := setupLogger(cfg)
	defer logger.Sync()

	logger.Info("Starting Exchange Rate Service",
		zap.String("environment", cfg.Environment),
		zap.Int("httpPort", cfg.HTTPPort),
		zap.Int("grpcPort", cfg.GRPCPort),
	)

	// Setup Redis client
	redisClient := setupRedis(cfg, logger)
	defer redisClient.Close()

	// Setup rate provider based on configuration
	rateProvider := setupProvider(cfg, logger)
	logger.Info("Rate provider configured", zap.String("provider", rateProvider.Name()))

	// Setup repository
	rateRepo := repository.NewRedisRepository(redisClient)

	// Setup metrics
	appMetrics := metrics.NewMetrics("exchange_rate_service")
	_ = appMetrics // Will be used when instrumenting handlers

	// Create rate service with dependency injection
	rateService := service.NewRateService(cfg, rateProvider, rateRepo, logger)

	// Setup Gin router
	router := setupRouter(cfg, logger, rateService, appMetrics)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create gRPC server
	grpcServer := setupGRPCServer(rateService, logger)

	// Start servers
	startServers(cfg, httpServer, grpcServer, logger)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Graceful shutdown
	shutdownServers(httpServer, grpcServer, redisClient, logger)

	logger.Info("Servers stopped")
}

func setupLogger(cfg *config.Config) *zap.Logger {
	var logger *zap.Logger
	var err error

	if cfg.IsProduction() {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic(err)
	}

	return logger
}

func setupRedis(cfg *config.Config, logger *zap.Logger) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		logger.Warn("Redis connection failed, caching disabled", zap.Error(err))
	} else {
		logger.Info("Connected to Redis", zap.String("addr", cfg.RedisAddr))
	}

	return redisClient
}

func setupProvider(cfg *config.Config, logger *zap.Logger) provider.RateProvider {
	switch cfg.ProviderType {
	case "simulated":
		providerCfg := provider.SimulatedProviderConfig{
			BaseSpread:           cfg.ProviderSpread,
			MaxDrift:             cfg.ProviderMaxDrift,
			RateValidityDuration: time.Duration(cfg.RateCacheTTL) * time.Second,
			DriftInterval:        5 * time.Second,
		}
		return provider.NewSimulatedProvider(providerCfg)

	// Future: Add real provider implementations
	// case "openexchangerates":
	//     return provider.NewOpenExchangeRatesProvider(cfg.OXRAppID, cfg.OXRAPIUrl)

	default:
		logger.Info("Unknown provider type, defaulting to simulated",
			zap.String("configured", cfg.ProviderType),
		)
		return provider.NewSimulatedProvider(provider.DefaultSimulatedConfig())
	}
}

func setupRouter(cfg *config.Config, logger *zap.Logger, rateService *service.RateService, appMetrics *metrics.Metrics) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	// Setup HTTP handler
	httpHandler := handler.NewHTTPHandler(rateService, logger)
	httpHandler.SetupRoutes(router)

	// Metrics endpoint
	if cfg.MetricsEnabled {
		router.GET(cfg.MetricsEndpoint, gin.WrapH(promhttp.Handler()))
	}

	return router
}

func setupGRPCServer(rateService *service.RateService, logger *zap.Logger) *grpc.Server {
	grpcServer := grpc.NewServer()

	// Register exchange rate service
	exchangeServer := grpcserver.NewExchangeRateServer(rateService, logger)
	grpcserver.RegisterExchangeRateServiceServer(grpcServer, exchangeServer)

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("exchange_rate_service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for debugging (disable in production if needed)
	reflection.Register(grpcServer)

	return grpcServer
}

func startServers(cfg *config.Config, httpServer *http.Server, grpcServer *grpc.Server, logger *zap.Logger) {
	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}

		logger.Info("Starting gRPC server", zap.Int("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()
}

func shutdownServers(httpServer *http.Server, grpcServer *grpc.Server, redisClient *redis.Client, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Gracefully stop gRPC server
	grpcServer.GracefulStop()

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		logger.Error("Redis close error", zap.Error(err))
	}
}

func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		)
	}
}
