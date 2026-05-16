package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// InstanceEvent is a normalized Kubernetes event for a workspace resource.
type InstanceEvent struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Object    string    `json:"object,omitempty"`
	Count     int32     `json:"count"`
	FirstTime time.Time `json:"firstTime"`
	LastTime  time.Time `json:"lastTime"`
}

// TerminalSession is metadata the backend uses to proxy WebSocket exec to the right pod.
// Token is a short-lived single-use credential — the browser passes it as ?token= on the
// WebSocket URL so it can connect without custom headers.
type TerminalSession struct {
	InstanceID    string `json:"instanceId"`
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	// ExecURL is the bridge WebSocket endpoint; append ?token=Token to connect.
	ExecURL string `json:"execUrl"`
	// Token is a 2-minute single-use auth token for the ExecURL WebSocket.
	Token string `json:"token"`
}

// RestartWorkspace triggers a rolling restart of the workspace Deployment by
// patching the pod template annotation (same mechanism as kubectl rollout restart).
// It then waits for the deployment to become ready and the health probe to pass.
func (b *Bridge) RestartInstance(ctx context.Context, workspaceID string) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	deployments, err := b.KubeClient.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	if len(deployments.Items) == 0 {
		return fmt.Errorf("no deployments found for workspace %s", workspaceID)
	}

	patch := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		time.Now().UTC().Format(time.RFC3339),
	)
	for _, dep := range deployments.Items {
		_, patchErr := b.KubeClient.AppsV1().Deployments(ns).Patch(
			ctx, dep.Name, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{},
		)
		if patchErr != nil {
			return fmt.Errorf("patch deployment %s: %w", dep.Name, patchErr)
		}
	}
	b.Logger.Printf("[RestartInstance] Rolling restart triggered for %s (ns=%s)", workspaceID, ns)

	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 5*time.Minute); err != nil {
		return err
	}
	return b.waitForInstanceHealth(ctx, workspaceID, 2*time.Minute)
}

// RedeployWorkspace re-runs a Helm upgrade using the spec stored in the release config.
// PVC, namespace, and release name are preserved; only the chart manifests are reapplied.
// OverwriteConfig is false so the agent's runtime config.yaml is not touched.
func (b *Bridge) RedeployInstance(ctx context.Context, workspaceID string) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.OverwriteConfig = false

	if _, err = b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[RedeployInstance] Redeployed %s", workspaceID)

	ns := b.workspaceNamespace(spec)
	releaseName := b.releaseName(workspaceID)
	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 5*time.Minute); err != nil {
		return err
	}
	return b.waitForInstanceHealth(ctx, workspaceID, 2*time.Minute)
}

// RollbackWorkspace rolls the Helm release back to a previous revision.
// version=0 means the immediately previous release (Helm default).
// Helm waits for the rollback to complete before returning.
func (b *Bridge) RollbackInstance(ctx context.Context, workspaceID string, version int) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	helmCfg, err := b.helmConfigForNamespace(ns)
	if err != nil {
		return fmt.Errorf("helm config: %w", err)
	}
	rollback := action.NewRollback(helmCfg)
	rollback.Version = version
	rollback.Wait = true
	rollback.Timeout = 5 * time.Minute
	if err := rollback.Run(rel.Name); err != nil {
		return fmt.Errorf("helm rollback: %w", err)
	}
	b.Logger.Printf("[RollbackInstance] Rolled back %s to version %d", workspaceID, version)
	return b.waitForInstanceHealth(ctx, workspaceID, 2*time.Minute)
}

// RepairWorkspace performs bounded auto-recovery based on the current pod state.
// Returns the action taken ("restart", "redeploy", or "none") and any error.
// Image pull failures are returned as errors — retrying would loop indefinitely.
func (b *Bridge) RepairInstance(ctx context.Context, workspaceID string) (string, error) {
	status, err := b.GetInstanceStatus(ctx, workspaceID)
	if err != nil {
		return "", err
	}

	switch status.WaitingReason {
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return "", fmt.Errorf("image pull failed; check image/tag and imagePullSecret")

	case "CrashLoopBackOff", "RunContainerError":
		if err := b.RestartInstance(ctx, workspaceID); err != nil {
			return "restart", err
		}
		return "restart", nil

	case "CreateContainerConfigError", "CreateContainerError":
		if err := b.RedeployInstance(ctx, workspaceID); err != nil {
			return "redeploy", err
		}
		return "redeploy", nil
	}

	if strings.EqualFold(status.PodPhase, string(corev1.PodFailed)) {
		if err := b.RedeployInstance(ctx, workspaceID); err != nil {
			return "redeploy", err
		}
		return "redeploy", nil
	}

	if !status.Healthy {
		if err := b.RestartInstance(ctx, workspaceID); err != nil {
			return "restart", err
		}
		return "restart", nil
	}

	return "none", nil
}

// GetInstanceEvents returns the 50 most recent Kubernetes events for resources
// belonging to this workspace release (deployment, pods, PVCs, service).
func (b *Bridge) GetInstanceEvents(ctx context.Context, workspaceID string) ([]InstanceEvent, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	// Build an allowlist of resource names owned by this release.
	relevant := map[string]bool{releaseName: true}
	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()

	pods, err := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range pods.Items {
			relevant[pods.Items[i].Name] = true
		}
	}
	pvcs, err := b.KubeClient.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range pvcs.Items {
			relevant[pvcs.Items[i].Name] = true
		}
	}
	replicasets, err := b.KubeClient.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range replicasets.Items {
			relevant[replicasets.Items[i].Name] = true
		}
	}

	list, err := b.KubeClient.CoreV1().Events(ns).List(ctx, metav1.ListOptions{Limit: 200})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	events := make([]InstanceEvent, 0, len(list.Items))
	for _, ev := range list.Items {
		if !relevant[ev.InvolvedObject.Name] {
			continue
		}
		lastTime := ev.LastTimestamp.Time
		if lastTime.IsZero() {
			lastTime = ev.EventTime.Time
		}
		events = append(events, InstanceEvent{
			Type:      ev.Type,
			Reason:    ev.Reason,
			Message:   ev.Message,
			Component: ev.Source.Component,
			Object:    fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
			Count:     ev.Count,
			FirstTime: ev.FirstTimestamp.Time,
			LastTime:  lastTime,
		})
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTime.After(events[j].LastTime)
	})
	if len(events) > 50 {
		events = events[:50]
	}
	return events, nil
}

// RecreateTerminalSession finds the best running pod for a terminal session.
// Returns metadata the backend uses to proxy exec — not a browser-usable URL.
func (b *Bridge) RecreateTerminalSession(ctx context.Context, workspaceID string) (*TerminalSession, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	pods, err := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	pod := selectPod(pods.Items)
	if pod == nil {
		return nil, fmt.Errorf("no running pods found for instance %s", workspaceID)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod %s is not running (phase: %s) — instance may still be starting", pod.Name, pod.Status.Phase)
	}

	containerName := ""
	for _, c := range pod.Spec.Containers {
		containerName = c.Name
		break
	}

	tok, err := b.issueExecToken(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("issue exec token: %w", err)
	}

	return &TerminalSession{
		InstanceID:   workspaceID,
		Namespace:     ns,
		PodName:       pod.Name,
		ContainerName: containerName,
		ExecURL:       fmt.Sprintf("/v1/instances/%s/exec", workspaceID),
		Token:         tok,
	}, nil
}

// deploymentReady returns true when the deployment's rollout is complete:
// all desired replicas are updated, ready, and available, and the observed
// generation has caught up. Returns an error string if the deployment has
// exceeded its ProgressDeadline (terminal failure).
func deploymentReady(dep *appsv1.Deployment) (bool, string) {
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Reason == "ProgressDeadlineExceeded" {
			return false, fmt.Sprintf("deployment %s exceeded progress deadline: %s", dep.Name, c.Message)
		}
	}
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if dep.Status.ObservedGeneration < dep.Generation {
		return false, ""
	}
	return dep.Status.UpdatedReplicas >= desired &&
		dep.Status.ReadyReplicas >= desired &&
		dep.Status.AvailableReplicas >= desired, ""
}

// waitForDeploymentReady polls until the deployment rollout is fully complete
// or the context deadline passes. Fails fast on ProgressDeadlineExceeded.
func (b *Bridge) waitForDeploymentReady(ctx context.Context, ns, releaseName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Check immediately before waiting for the first tick.
	if dep, err := b.KubeClient.AppsV1().Deployments(ns).Get(ctx, releaseName, metav1.GetOptions{}); err == nil {
		if ok, fatal := deploymentReady(dep); fatal != "" {
			return fmt.Errorf("%s", fatal)
		} else if ok {
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("deployment %s/%s did not become ready within %s", ns, releaseName, timeout)
		case <-ticker.C:
			dep, err := b.KubeClient.AppsV1().Deployments(ns).Get(ctx, releaseName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if ok, fatal := deploymentReady(dep); fatal != "" {
				return fmt.Errorf("%s", fatal)
			} else if ok {
				return nil
			}
		}
	}
}

// waitForWorkspaceHealth polls the internal health probe until it returns 2xx or the context deadline passes.
func (b *Bridge) waitForInstanceHealth(ctx context.Context, workspaceID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec for health wait: %w", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Check immediately before waiting for the first tick.
	if healthy, _, _ := b.checkInstanceHealth(ctx, spec); healthy {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("instance %s did not become healthy within %s", workspaceID, timeout)
		case <-ticker.C:
			if healthy, _, _ := b.checkInstanceHealth(ctx, spec); healthy {
				return nil
			}
		}
	}
}
