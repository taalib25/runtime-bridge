package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const bridgeStateConfigMap = "hermes-bridge-state"

// loadMaintenanceState restores the maintenance flag from the persisted ConfigMap.
// Non-fatal: a missing or unreadable ConfigMap leaves maintenance=false (the safe default).
func (b *Bridge) loadMaintenanceState(ctx context.Context) {
	if b.KubeClient == nil {
		return
	}
	cm, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Get(ctx, bridgeStateConfigMap, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			b.Logger.Printf("[Maintenance] WARNING: could not read state ConfigMap: %v", err)
		}
		return
	}
	if cm.Data["maintenance"] == "true" {
		b.maintenance.Store(true)
		b.Logger.Printf("[Maintenance] restored from ConfigMap: mode=on")
	}
}

// saveMaintenanceState persists the maintenance flag to a ConfigMap so that the
// state survives pod restarts. Non-fatal: a write failure is logged but does not
// affect the in-memory flag (the UI will still see the correct live state).
func (b *Bridge) saveMaintenanceState(ctx context.Context, enabled bool) {
	if b.KubeClient == nil {
		return
	}
	value := "false"
	if enabled {
		value = "true"
	}

	existing, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Get(ctx, bridgeStateConfigMap, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bridgeStateConfigMap,
				Namespace: b.Config.Namespace,
				Labels:    map[string]string{"app.kubernetes.io/managed-by": "hermes-bridge"},
			},
			Data: map[string]string{"maintenance": value},
		}
		if _, cerr := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Create(ctx, cm, metav1.CreateOptions{}); cerr != nil {
			b.Logger.Printf("[Maintenance] WARNING: could not create state ConfigMap: %v", cerr)
		}
		return
	}
	if err != nil {
		b.Logger.Printf("[Maintenance] WARNING: could not read state ConfigMap before update: %v", err)
		return
	}
	if existing.Data == nil {
		existing.Data = make(map[string]string)
	}
	existing.Data["maintenance"] = value
	if _, uerr := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Update(ctx, existing, metav1.UpdateOptions{}); uerr != nil {
		b.Logger.Printf("[Maintenance] WARNING: could not update state ConfigMap: %v", uerr)
	}
}
