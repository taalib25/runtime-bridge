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
	workspaces, err := b.ListInstances(ctx)
	if err != nil {
		b.Logger.Printf("Failed to list instances during sync: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, workspace := range workspaces {
		health := 0.0
		if workspace.Healthy {
			health = 1.0
		}
		b.Metrics.WorkspaceHealth.WithLabelValues(b.ClusterName, workspace.InstanceID).Set(health)

		if !workspace.Healthy || workspace.Phase == "failed" {
			b.Logger.Printf("ALERT instance unhealthy: id=%s phase=%s namespace=%s message=%s", workspace.InstanceID, workspace.Phase, workspace.Namespace, workspace.Message)
			continue
		}
		if workspace.Phase == "creating" && now.Sub(workspace.CreatedAt) > b.Config.OperationTimeout {
			b.Logger.Printf("ALERT instance stuck creating: id=%s namespace=%s age=%s", workspace.InstanceID, workspace.Namespace, now.Sub(workspace.CreatedAt))
		}
	}
}
