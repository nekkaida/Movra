package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the exchange rate service
type Metrics struct {
	// Request metrics
	RateRequestsTotal    *prometheus.CounterVec
	RateRequestDuration  *prometheus.HistogramVec

	// Cache metrics
	CacheHitsTotal       *prometheus.CounterVec
	CacheMissesTotal     *prometheus.CounterVec

	// Lock metrics
	LockedRatesActive    prometheus.Gauge
	RateLockDuration     *prometheus.HistogramVec

	// Provider metrics
	ProviderRequestsTotal   *prometheus.CounterVec
	ProviderErrorsTotal     *prometheus.CounterVec
	ProviderRequestDuration *prometheus.HistogramVec

	// Business metrics
	QuotesGeneratedTotal *prometheus.CounterVec
}

// NewMetrics creates and registers all metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "exchange_rate_service"
	}

	return &Metrics{
		RateRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_requests_total",
				Help:      "Total number of rate requests",
			},
			[]string{"source_currency", "target_currency", "status"},
		),

		RateRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "rate_request_duration_seconds",
				Help:      "Duration of rate requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"source_currency", "target_currency", "cache_hit"},
		),

		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type"}, // "rate" or "locked_rate"
		),

		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type"},
		),

		LockedRatesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "locked_rates_active",
				Help:      "Number of currently active rate locks",
			},
		),

		RateLockDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "rate_lock_duration_seconds",
				Help:      "Duration of rate locks in seconds",
				Buckets:   []float64{15, 30, 45, 60, 90, 120},
			},
			[]string{"source_currency", "target_currency"},
		),

		ProviderRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "provider_requests_total",
				Help:      "Total number of requests to rate providers",
			},
			[]string{"provider", "status"},
		),

		ProviderErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "provider_errors_total",
				Help:      "Total number of errors from rate providers",
			},
			[]string{"provider", "error_type"},
		),

		ProviderRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "provider_request_duration_seconds",
				Help:      "Duration of provider requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"provider"},
		),

		QuotesGeneratedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "quotes_generated_total",
				Help:      "Total number of quotes generated",
			},
			[]string{"source_currency", "target_currency"},
		),
	}
}

// RecordRateRequest records metrics for a rate request
func (m *Metrics) RecordRateRequest(source, target, status string, durationSeconds float64, cacheHit bool) {
	m.RateRequestsTotal.WithLabelValues(source, target, status).Inc()

	cacheHitStr := "false"
	if cacheHit {
		cacheHitStr = "true"
	}
	m.RateRequestDuration.WithLabelValues(source, target, cacheHitStr).Observe(durationSeconds)
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.CacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.CacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// RecordRateLock records a rate lock event
func (m *Metrics) RecordRateLock(source, target string, durationSeconds float64) {
	m.LockedRatesActive.Inc()
	m.RateLockDuration.WithLabelValues(source, target).Observe(durationSeconds)
}

// RecordRateLockExpired records when a rate lock expires or is released
func (m *Metrics) RecordRateLockExpired() {
	m.LockedRatesActive.Dec()
}

// RecordProviderRequest records a provider request
func (m *Metrics) RecordProviderRequest(provider, status string, durationSeconds float64) {
	m.ProviderRequestsTotal.WithLabelValues(provider, status).Inc()
	m.ProviderRequestDuration.WithLabelValues(provider).Observe(durationSeconds)
}

// RecordProviderError records a provider error
func (m *Metrics) RecordProviderError(provider, errorType string) {
	m.ProviderErrorsTotal.WithLabelValues(provider, errorType).Inc()
}

// RecordQuoteGenerated records a quote generation
func (m *Metrics) RecordQuoteGenerated(source, target string) {
	m.QuotesGeneratedTotal.WithLabelValues(source, target).Inc()
}
