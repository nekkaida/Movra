package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	port := getEnv("HTTP_PORT", "8083")

	// Setup Gin
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

	// API endpoints
	api := router.Group("/api")
	{
		payouts := api.Group("/payouts")
		{
			payouts.POST("/", initiatePayoutHandler)
			payouts.GET("/:id", getPayoutHandler)
			payouts.POST("/:id/retry", retryPayoutHandler)
			payouts.POST("/:id/cancel", cancelPayoutHandler)
		}
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start HTTP server
	go func() {
		logger.Info("Starting Settlement Service", zap.String("port", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}

func initiatePayoutHandler(c *gin.Context) {
	// TODO: Implement payout initiation
	c.JSON(http.StatusCreated, gin.H{
		"id":       "payout_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"status":   "PENDING",
		"message":  "Payout initiated",
	})
}

func getPayoutHandler(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "PROCESSING",
	})
}

func retryPayoutHandler(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "PENDING",
		"retryCount": 1,
	})
}

func cancelPayoutHandler(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "CANCELLED",
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
