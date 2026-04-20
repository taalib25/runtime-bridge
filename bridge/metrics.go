package main

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	WorkspaceCount   *prometheus.GaugeVec
	OperationLatency *prometheus.HistogramVec
	OperationResults *prometheus.CounterVec
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
	}

	registerer.MustRegister(m.WorkspaceCount, m.OperationLatency, m.OperationResults)
	m.WorkspaceCount.WithLabelValues(clusterName).Set(0)
	return m
}
