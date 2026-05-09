package main

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	WorkspaceCount   *prometheus.GaugeVec
	OperationLatency *prometheus.HistogramVec
	OperationResults *prometheus.CounterVec
	WorkspaceHealth  *prometheus.GaugeVec
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
}

func NewMetrics(clusterName string, registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		WorkspaceCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hermes_bridge",
			Name:      "workspace_count",
			Help:      "Number of Helm-managed workspaces visible to the bridge",
		}, []string{"cluster"}),
		OperationLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hermes_bridge",
			Name:      "operation_latency_seconds",
			Help:      "Workspace operation latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"cluster", "operation", "result"}),
		OperationResults: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "hermes_bridge",
			Name:      "operation_total",
			Help:      "Total workspace operations grouped by result",
		}, []string{"cluster", "operation", "result"}),
		WorkspaceHealth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hermes_bridge",
			Name:      "workspace_health",
			Help:      "Workspace HTTP health check result (1=healthy, 0=unhealthy)",
		}, []string{"cluster", "workspace_id"}),
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
		m.WorkspaceCount, m.OperationLatency, m.OperationResults,
		m.WorkspaceHealth, m.HTTPRequests, m.HTTPDuration,
	)
	m.WorkspaceCount.WithLabelValues(clusterName).Set(0)
	return m
}
