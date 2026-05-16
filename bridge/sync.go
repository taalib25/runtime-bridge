package main

import (
	"context"
	"time"
)

func (b *Bridge) StartSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(b.Config.SyncInterval)
	defer ticker.Stop()

	b.syncInstances(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.syncInstances(ctx)
		}
	}
}

func (b *Bridge) syncInstances(ctx context.Context) {
	instances, err := b.ListInstances(ctx)
	if err != nil {
		b.Logger.Printf("Failed to list instances during sync: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, instance := range instances {
		health := 0.0
		if instance.Healthy {
			health = 1.0
		}
		b.Metrics.InstanceHealth.WithLabelValues(b.ClusterName, instance.InstanceID).Set(health)

		if !instance.Healthy || instance.Phase == "failed" {
			b.Logger.Printf("ALERT instance unhealthy: id=%s phase=%s namespace=%s message=%s", instance.InstanceID, instance.Phase, instance.Namespace, instance.Message)
			continue
		}
		if instance.Phase == "creating" && now.Sub(instance.CreatedAt) > b.Config.OperationTimeout {
			b.Logger.Printf("ALERT instance stuck creating: id=%s namespace=%s age=%s", instance.InstanceID, instance.Namespace, now.Sub(instance.CreatedAt))
		}
	}
}
