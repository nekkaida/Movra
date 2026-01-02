package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/movra/exchange-rate-service/internal/model"
	"github.com/movra/exchange-rate-service/internal/service"
	"go.uber.org/zap"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	rateService *service.RateService
	logger      *zap.Logger
}

// NewHTTPHandler creates a new HTTPHandler
func NewHTTPHandler(rateService *service.RateService, logger *zap.Logger) *HTTPHandler {
	return &HTTPHandler{
		rateService: rateService,
		logger:      logger,
	}
}

// SetupRoutes configures the HTTP routes
func (h *HTTPHandler) SetupRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// Metrics
	r.GET("/metrics", h.Metrics)

	// Rate endpoints
	api := r.Group("/api")
	{
		rates := api.Group("/rates")
		{
			rates.GET("/:from/:to", h.GetRate)
			rates.POST("/lock", h.LockRate)
			rates.GET("/locked/:lockId", h.GetLockedRate)
		}
		api.GET("/corridors", h.GetCorridors)
	}
}

// Health returns the health status
func (h *HTTPHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "exchange-rate-service",
	})
}

// Ready returns the readiness status
func (h *HTTPHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "exchange-rate-service",
	})
}

// Metrics returns Prometheus metrics (placeholder)
func (h *HTTPHandler) Metrics(c *gin.Context) {
	// In production, use promhttp.Handler()
	c.String(http.StatusOK, "# Prometheus metrics endpoint")
}

// GetRate retrieves the current exchange rate
func (h *HTTPHandler) GetRate(c *gin.Context) {
	from := c.Param("from")
	to := c.Param("to")

	if len(from) != 3 || len(to) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency code format"})
		return
	}

	rate, err := h.rateService.GetRate(c.Request.Context(), from, to)
	if err != nil {
		h.logger.Error("Failed to get rate", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rate)
}

// LockRate locks a rate for a transfer
func (h *HTTPHandler) LockRate(c *gin.Context) {
	var req model.RateLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	locked, err := h.rateService.LockRate(c.Request.Context(), req.SourceCurrency, req.TargetCurrency, req.DurationSeconds)
	if err != nil {
		h.logger.Error("Failed to lock rate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, locked)
}

// GetLockedRate retrieves a previously locked rate
func (h *HTTPHandler) GetLockedRate(c *gin.Context) {
	lockID := c.Param("lockId")

	locked, err := h.rateService.GetLockedRate(c.Request.Context(), lockID)
	if err != nil {
		h.logger.Error("Failed to get locked rate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if locked.Expired {
		c.JSON(http.StatusGone, gin.H{"error": "Rate lock expired", "expired": true})
		return
	}

	c.JSON(http.StatusOK, locked)
}

// GetCorridors returns available corridors
func (h *HTTPHandler) GetCorridors(c *gin.Context) {
	sourceCurrency := c.Query("source")
	corridors := h.rateService.GetCorridors(sourceCurrency)
	c.JSON(http.StatusOK, gin.H{"corridors": corridors})
}
