package main

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	InstanceCount   *prometheus.GaugeVec
	OperationLatency *prometheus.HistogramVec
	OperationResults *prometheus.CounterVec
	InstanceHealth  *prometheus.GaugeVec
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
}

func NewMetrics(clusterName string, registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		InstanceCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hermes_bridge",
			Name:      "instance_count",
			Help:      "Number of Helm-managed instances visible to the bridge",
		}, []string{"cluster"}),
		OperationLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hermes_bridge",
			Name:      "operation_latency_seconds",
			Help:      "Instance operation latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"cluster", "operation", "result"}),
		OperationResults: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "hermes_bridge",
			Name:      "operation_total",
			Help:      "Total instance operations grouped by result",
		}, []string{"cluster", "operation", "result"}),
		InstanceHealth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hermes_bridge",
			Name:      "instance_health",
			Help:      "Instance HTTP health check result (1=healthy, 0=unhealthy)",
		}, []string{"cluster", "instance_id"}),
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "hermes_bridge",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled by the bridge",
		}, []string{"method", "path", "status_code"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hermes_bridge",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "path"}),
	}

	registerer.MustRegister(
		m.InstanceCount, m.OperationLatency, m.OperationResults,
		m.InstanceHealth, m.HTTPRequests, m.HTTPDuration,
	)
	m.InstanceCount.WithLabelValues(clusterName).Set(0)
	return m
}
