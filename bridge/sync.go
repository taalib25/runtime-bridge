package main

import (
	"context"
	"time"
)

func (b *Bridge) StartSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(b.Config.SyncInterval)
	defer ticker.Stop()

	b.syncWorkspaces(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.syncWorkspaces(ctx)
		}
	}
}

func (b *Bridge) syncWorkspaces(ctx context.Context) {
	workspaces, err := b.ListWorkspaces(ctx)
	if err != nil {
		b.Logger.Printf("Failed to list workspaces during sync: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, workspace := range workspaces {
		if !workspace.Healthy || workspace.Phase == "failed" {
			b.Logger.Printf("ALERT workspace unhealthy: id=%s phase=%s namespace=%s message=%s", workspace.WorkspaceID, workspace.Phase, workspace.Namespace, workspace.Message)
			continue
		}
		if workspace.Phase == "creating" && now.Sub(workspace.CreatedAt) > b.Config.OperationTimeout {
			b.Logger.Printf("ALERT workspace stuck creating: id=%s namespace=%s age=%s", workspace.WorkspaceID, workspace.Namespace, now.Sub(workspace.CreatedAt))
		}
	}
}
