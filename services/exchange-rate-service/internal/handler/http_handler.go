package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/service"
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
		api.GET("/quote", h.GetQuote)
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
	// Check if service dependencies are healthy
	if err := h.rateService.Health(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"service": "exchange-rate-service",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "exchange-rate-service",
	})
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

// GetQuote generates a rate quote with fees
func (h *HTTPHandler) GetQuote(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	amountStr := c.Query("amount")

	if from == "" || to == "" || amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "from, to, and amount query parameters are required",
		})
		return
	}

	if len(from) != 3 || len(to) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency code format"})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	quote, err := h.rateService.GetQuote(c.Request.Context(), from, to, amount)
	if err != nil {
		h.logger.Error("Failed to get quote",
			zap.String("from", from),
			zap.String("to", to),
			zap.Float64("amount", amount),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quote)
}
