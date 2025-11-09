package services

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// AdMetrics provides Prometheus metrics collection for ad conversions
type AdMetrics struct {
	conversionsTotal   *prometheus.CounterVec
	rewardsTotal       *prometheus.CounterVec
	fraudAttemptsTotal *prometheus.CounterVec
	callbackLatency    *prometheus.HistogramVec
	conversionsByTier  *prometheus.CounterVec
}

// NewAdMetrics creates a new ad metrics collector
func NewAdMetrics(
	conversionsTotal *prometheus.CounterVec,
	rewardsTotal *prometheus.CounterVec,
	fraudAttemptsTotal *prometheus.CounterVec,
	callbackLatency *prometheus.HistogramVec,
	conversionsByTier *prometheus.CounterVec,
) *AdMetrics {
	return &AdMetrics{
		conversionsTotal:   conversionsTotal,
		rewardsTotal:       rewardsTotal,
		fraudAttemptsTotal: fraudAttemptsTotal,
		callbackLatency:    callbackLatency,
		conversionsByTier:  conversionsByTier,
	}
}

// RecordConversion records a successful conversion
func (m *AdMetrics) RecordConversion(provider, adType, status string, reward int, tier string) {
	if m.conversionsTotal != nil {
		m.conversionsTotal.WithLabelValues(provider, adType, status).Inc()
	}
	
	if status == "completed" && m.rewardsTotal != nil {
		m.rewardsTotal.WithLabelValues(provider, adType).Add(float64(reward))
	}
	
	if m.conversionsByTier != nil {
		m.conversionsByTier.WithLabelValues(tier).Inc()
	}
}

// RecordFraud records a fraud attempt
func (m *AdMetrics) RecordFraud(provider, reason string) {
	if m.fraudAttemptsTotal != nil {
		m.fraudAttemptsTotal.WithLabelValues(provider, reason).Inc()
	}
}

// RecordLatency records callback processing latency
func (m *AdMetrics) RecordLatency(provider, status string, duration time.Duration) {
	if m.callbackLatency != nil {
		m.callbackLatency.WithLabelValues(provider, status).Observe(duration.Seconds())
	}
}

// ObserveCallbackLatency wraps callback execution and records latency
func (m *AdMetrics) ObserveCallbackLatency(provider string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.RecordLatency(provider, status, duration)
	return err
}
