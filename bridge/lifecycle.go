package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

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

// RestartInstance triggers a rolling restart of the instance Deployment by
// patching the pod template annotation (same mechanism as kubectl rollout restart).
// It then waits for the deployment to become ready and the health probe to pass.
func (b *Bridge) RestartInstance(ctx context.Context, instanceID string) error {
	rel, err := b.lookupRelease(ctx, instanceID)
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
		return fmt.Errorf("no deployments found for instance %s", instanceID)
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
	b.Logger.Printf("[RestartInstance] Rolling restart triggered for %s (ns=%s)", instanceID, ns)

	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 6*time.Minute); err != nil {
		return err
	}
	return b.waitForInstanceHealth(ctx, instanceID, 2*time.Minute)
}

// RedeployInstance re-runs a Helm upgrade using the spec stored in the release config.
// PVC, namespace, and release name are preserved; only the chart manifests are reapplied.
// OverwriteConfig is false so the agent's runtime config.yaml is not touched.
func (b *Bridge) RedeployInstance(ctx context.Context, instanceID string) error {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return err
	}
	spec, err := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.OverwriteConfig = false

	if _, err = b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[RedeployInstance] Redeployed %s", instanceID)

	ns := b.instanceNamespace(spec)
	releaseName := b.releaseName(instanceID)
	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 6*time.Minute); err != nil {
		return err
	}
	return b.waitForInstanceHealth(ctx, instanceID, 2*time.Minute)
}

// UpgradeInstance upgrades the running image to the specified image/tag via Helm upgrade.
// On health check failure it automatically rolls back to the previous Helm revision.
// Returns the final image string ("repo:tag") on success.
func (b *Bridge) UpgradeInstance(ctx context.Context, instanceID, image, imageTag string) (string, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return "", err
	}
	previousRevision := rel.Version

	spec, err := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return "", fmt.Errorf("reconstruct spec: %w", err)
	}
	if image != "" {
		spec.Image = image
	}
	spec.ImageTag = imageTag
	spec.OverwriteConfig = false

	if _, err = b.UpdateInstance(ctx, spec); err != nil {
		return "", fmt.Errorf("helm upgrade: %w", err)
	}

	ns := b.instanceNamespace(spec)
	releaseName := b.releaseName(instanceID)

	rollback := func(reason error) error {
		// If the operation was canceled (a delete superseded it), the release is
		// being removed — rolling back would race the uninstall. Bail cleanly.
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("upgrade superseded by delete")
		}
		// Roll back under a fresh context: the operation deadline may already be
		// drained by the failed upgrade, which would make the rollback's health
		// wait fail instantly and falsely report "rollback also failed".
		rbCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		defer cancel()
		b.Logger.Printf("[UpgradeInstance] %s unhealthy after upgrade, rolling back to revision %d: %v", instanceID, previousRevision, reason)
		if rbErr := b.RollbackInstance(rbCtx, instanceID, previousRevision); rbErr != nil {
			return fmt.Errorf("upgrade failed: %v; rollback also failed: %w", reason, rbErr)
		}
		return fmt.Errorf("upgrade failed: %v — rolled back to revision %d", reason, previousRevision)
	}

	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 6*time.Minute); err != nil {
		return "", rollback(err)
	}
	if err := b.waitForInstanceHealth(ctx, instanceID, 2*time.Minute); err != nil {
		return "", rollback(err)
	}

	finalImage := fmt.Sprintf("%s:%s", spec.Image, imageTag)
	b.Logger.Printf("[UpgradeInstance] Upgraded %s to %s", instanceID, finalImage)
	return fmt.Sprintf("upgrade completed: %s", finalImage), nil
}

// RollbackInstance rolls the Helm release back to a previous revision.
// version=0 means the immediately previous release (Helm default).
// Helm waits for the rollback to complete before returning.
func (b *Bridge) RollbackInstance(ctx context.Context, instanceID string, version int) error {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	helmCfg, err := b.helmConfigForNamespace(ns)
	if err != nil {
		return fmt.Errorf("helm config: %w", err)
	}
	rollbackOpts := RollbackOptions{Version: version, Wait: true, Timeout: 5 * time.Minute}
	if err := b.Helm.Rollback(helmCfg, rollbackOpts, rel.Name); err != nil {
		return fmt.Errorf("helm rollback: %w", err)
	}
	b.Logger.Printf("[RollbackInstance] Rolled back %s to version %d", instanceID, version)
	return b.waitForInstanceHealth(ctx, instanceID, 2*time.Minute)
}

// RepairInstance performs bounded auto-recovery based on the current pod state.
// Returns the action taken ("restart", "redeploy", or "none") and any error.
// Unrecoverable conditions (image pull, unschedulable, node pressure eviction)
// are returned as errors — retrying these would loop indefinitely.
func (b *Bridge) RepairInstance(ctx context.Context, instanceID string) (string, error) {
	status, err := b.GetInstanceStatus(ctx, instanceID)
	if err != nil {
		return "", err
	}

	// --- Unrecoverable: bridge cannot fix these without external action ---
	switch status.WaitingReason {
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return "", fmt.Errorf("image pull failed; check image/tag and imagePullSecret")
	case "Unschedulable":
		return "", fmt.Errorf("pod is unschedulable: cluster may be full or all nodes are cordoned")
	}

	// --- Check for pod-level eviction (phase=Failed with reason=Evicted) ---
	// Eviction happens under node memory/disk pressure — must redeploy so
	// Kubernetes can schedule the pod on a healthy node.
	if strings.EqualFold(status.PodPhase, string(corev1.PodFailed)) {
		if podEvicted, msg := b.isPodEvicted(ctx, instanceID); podEvicted {
			b.Logger.Printf("[RepairInstance] pod evicted for %s (%s), redeploying", instanceID, msg)
			if err := b.RedeployInstance(ctx, instanceID); err != nil {
				return "redeploy", err
			}
			return "redeploy", nil
		}
		// Other Failed phase (e.g. ContainerCannotRun at pod level) → redeploy.
		if err := b.RedeployInstance(ctx, instanceID); err != nil {
			return "redeploy", err
		}
		return "redeploy", nil
	}

	// --- Pod exited cleanly: Succeeded means the agent process stopped, restart it ---
	if strings.EqualFold(status.PodPhase, string(corev1.PodSucceeded)) {
		b.Logger.Printf("[RepairInstance] pod phase Succeeded for %s (agent exited cleanly), restarting", instanceID)
		if err := b.RestartInstance(ctx, instanceID); err != nil {
			return "restart", err
		}
		return "restart", nil
	}

	// --- Unknown phase: node lost contact, redeploy to reschedule ---
	if strings.EqualFold(status.PodPhase, string(corev1.PodUnknown)) {
		b.Logger.Printf("[RepairInstance] pod phase Unknown for %s (node may be unreachable), redeploying", instanceID)
		if err := b.RedeployInstance(ctx, instanceID); err != nil {
			return "redeploy", err
		}
		return "redeploy", nil
	}

	// --- Container waiting reasons ---
	switch status.WaitingReason {
	case "CrashLoopBackOff", "RunContainerError", "PostStartHookError":
		if err := b.RestartInstance(ctx, instanceID); err != nil {
			return "restart", err
		}
		return "restart", nil

	case "CreateContainerConfigError", "CreateContainerError", "ContainerCannotRun":
		// Config or runtime setup is broken — re-running Helm may fix a bad
		// secret reference or an incorrect pod spec field.
		if err := b.RedeployInstance(ctx, instanceID); err != nil {
			return "redeploy", err
		}
		return "redeploy", nil

	case "OOMKilled":
		// Container was OOM-killed in its last run; it may be in backoff now.
		// A restart gives it a fresh memory slate.
		if err := b.RestartInstance(ctx, instanceID); err != nil {
			return "restart", err
		}
		return "restart", nil
	}

	// --- Init container stuck (WaitingReason is "Init:<reason>") ---
	if strings.HasPrefix(status.WaitingReason, "Init:") {
		initReason := strings.TrimPrefix(status.WaitingReason, "Init:")
		switch initReason {
		case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
			return "", fmt.Errorf("init container image pull failed; check image/tag and imagePullSecret")
		default:
			// Init container is crashing or misconfigured — redeploy to reapply
			// the chart which may correct a misconfigured init container spec.
			b.Logger.Printf("[RepairInstance] init container stuck (%s) for %s, redeploying", initReason, instanceID)
			if err := b.RedeployInstance(ctx, instanceID); err != nil {
				return "redeploy", err
			}
			return "redeploy", nil
		}
	}

	// --- Terminated container: check the current terminated reason directly ---
	if terminatedReason := b.containerTerminatedReason(ctx, instanceID); terminatedReason != "" {
		switch terminatedReason {
		case "OOMKilled":
			if err := b.RestartInstance(ctx, instanceID); err != nil {
				return "restart", err
			}
			return "restart", nil
		case "ContainerCannotRun":
			if err := b.RedeployInstance(ctx, instanceID); err != nil {
				return "redeploy", err
			}
			return "redeploy", nil
		default:
			// "Error", "Completed", or anything else — container exited unexpectedly.
			if err := b.RestartInstance(ctx, instanceID); err != nil {
				return "restart", err
			}
			return "restart", nil
		}
	}

	// --- Fallback: unhealthy but no specific condition detected → try restart ---
	if !status.Healthy {
		if err := b.RestartInstance(ctx, instanceID); err != nil {
			return "restart", err
		}
		return "restart", nil
	}

	return "none", nil
}

// isPodEvicted checks whether the pod for the given workspace was evicted by the
// kubelet (e.g. due to node disk/memory pressure). Returns the eviction message.
func (b *Bridge) isPodEvicted(ctx context.Context, instanceID string) (bool, string) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return false, ""
	}
	selector := labels.Set{"app.kubernetes.io/instance": rel.Name}.AsSelector().String()
	pods, err := b.KubeClient.CoreV1().Pods(rel.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, ""
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "Evicted" {
			return true, pod.Status.Message
		}
	}
	return false, ""
}

// containerTerminatedReason returns the Terminated.Reason of the first container
// that is currently in the Terminated state (not last termination). Returns ""
// when no container is terminated right now.
func (b *Bridge) containerTerminatedReason(ctx context.Context, instanceID string) string {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return ""
	}
	selector := labels.Set{"app.kubernetes.io/instance": rel.Name}.AsSelector().String()
	pods, err := b.KubeClient.CoreV1().Pods(rel.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return ""
	}
	pod := selectPod(pods.Items)
	if pod == nil {
		return ""
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}
	return ""
}

// RecreateTerminalSession finds the best running pod for a terminal session.
// Returns metadata the backend uses to proxy exec — not a browser-usable URL.
func (b *Bridge) RecreateTerminalSession(ctx context.Context, instanceID string) (*TerminalSession, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
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
		return nil, fmt.Errorf("no running pods found for instance %s", instanceID)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod %s is not running (phase: %s) — instance may still be starting", pod.Name, pod.Status.Phase)
	}

	containerName := ""
	for _, c := range pod.Spec.Containers {
		containerName = c.Name
		break
	}

	tok, err := b.issueExecToken(instanceID)
	if err != nil {
		return nil, fmt.Errorf("issue exec token: %w", err)
	}

	return &TerminalSession{
		InstanceID:    instanceID,
		Namespace:     ns,
		PodName:       pod.Name,
		ContainerName: containerName,
		ExecURL:       fmt.Sprintf("/v1/instances/%s/exec", instanceID),
		Token:         tok,
	}, nil
}
