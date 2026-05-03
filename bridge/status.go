package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (b *Bridge) GetWorkspaceStatus(ctx context.Context, workspaceID string) (WorkspaceStatus, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return b.collectWorkspaceStatus(ctx, spec, rel.Name, rel.Info.FirstDeployed.Time)
}

func (b *Bridge) collectWorkspaceStatus(ctx context.Context, spec WorkspaceSpec, releaseName string, createdAt time.Time) (WorkspaceStatus, error) {
	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	namespace := b.workspaceNamespace(spec)

	deployments, err := b.KubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return WorkspaceStatus{}, fmt.Errorf("list deployments: %w", err)
	}
	pods, err := b.KubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return WorkspaceStatus{}, fmt.Errorf("list pods: %w", err)
	}

	status := WorkspaceStatus{
		WorkspaceID:   spec.WorkspaceID,
		ClusterID:     spec.ClusterID,
		ReleaseName:   releaseName,
		Namespace:     namespace,
		URL:           workspaceURL(spec),
		DashboardURL:  dashboardURL(spec),
		CreatedAt:     createdAt.UTC(),
		LastCheckedAt: time.Now().UTC(),
		Spec:          spec,
	}

	if dep := selectDeployment(deployments.Items, releaseName); dep != nil {
		status.Replicas = dep.Status.Replicas
		status.ReadyReplicas = dep.Status.ReadyReplicas
		status.Conditions = deploymentConditions(convertDeploymentConditions(dep.Status.Conditions))
		status.Message = deploymentMessage(*dep)
	}

	if pod := selectPod(pods.Items); pod != nil {
		status.PodPhase = string(pod.Status.Phase)
		status.WaitingReason = containerWaitingReason(pod)
		status.RestartCount = podRestartCount(pod)
		if oomKilled(pod) && status.WaitingReason == "" {
			status.WaitingReason = "OOMKilled"
		}
		if message := podMessage(*pod); message != "" {
			status.Message = message
		}
	}

	healthy, code, healthErr := b.checkWorkspaceHealth(ctx, spec)
	status.Healthy = healthy
	status.HealthStatusCode = code
	if healthErr != nil && status.Message == "" {
		status.Message = healthErr.Error()
	}
	status.Phase = derivePhase(status, healthErr)
	return status, nil
}

// containerWaitingReason returns the Waiting.Reason of the first container not
// yet running. Common values: CrashLoopBackOff, ImagePullBackOff, ErrImagePull,
// CreateContainerConfigError, CreateContainerError, RunContainerError.
func containerWaitingReason(pod *corev1.Pod) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
	}
	// Also check init containers — a stuck init container blocks everything.
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return "Init:" + cs.State.Waiting.Reason
		}
	}
	return ""
}

// podRestartCount returns the total restart count across all containers.
func podRestartCount(pod *corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// oomKilled returns true if any container was last terminated due to OOM.
func oomKilled(pod *corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.LastTerminationState.Terminated != nil &&
			cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return true
		}
	}
	return false
}

func (b *Bridge) checkWorkspaceHealth(ctx context.Context, spec WorkspaceSpec) (bool, int, error) {
	// Probe the internal Kubernetes service directly so health checks bypass
	// Traefik ForwardAuth — the pod is healthy even before the auth endpoint exists.
	url := workspaceInternalHealthURL(spec, b.Config.HealthPath)
	if url == "" {
		return false, 0, fmt.Errorf("workspace URL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0, err
	}
	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, resp.StatusCode, nil
}

func derivePhase(status WorkspaceStatus, healthErr error) string {
	// Pod-level hard failure (evicted, node issues, OOM at pod level).
	if strings.EqualFold(status.PodPhase, string(corev1.PodFailed)) {
		return "failed"
	}

	// Container-level error states — pod may be Running or Pending but the
	// container is definitively broken and needs user intervention.
	switch status.WaitingReason {
	case "CrashLoopBackOff", "RunContainerError", "PostStartHookError",
		"CreateContainerError", "CreateContainerConfigError":
		return "error"
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return "error"
	}
	if status.WaitingReason == "OOMKilled" {
		return "error"
	}

	// Fully healthy.
	if status.ReadyReplicas > 0 && status.Healthy && healthErr == nil {
		return "ready"
	}

	// Pod is Running but app hasn't passed health check yet — still warming up.
	// Do NOT return "failed" here: health check failure during startup is expected.
	if strings.EqualFold(status.PodPhase, string(corev1.PodRunning)) {
		return "starting"
	}

	// Not yet scheduled, waiting for resources, or init containers still running.
	return "creating"
}

func dashboardHost(spec WorkspaceSpec) string {
	host := spec.Network.host()
	if host == "" {
		return ""
	}
	return "dash-" + host
}

func dashboardURL(spec WorkspaceSpec) string {
	h := dashboardHost(spec)
	if h == "" {
		return ""
	}
	scheme := spec.Network.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + h
}

func workspaceURL(spec WorkspaceSpec) string {
	host := spec.Network.host()
	if strings.TrimSpace(host) == "" {
		return ""
	}
	scheme := spec.Network.Scheme
	if scheme == "" {
		scheme = "https"
	}
	path := spec.Network.path()
	if path == "/" {
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func resolveHealthPath(spec WorkspaceSpec, defaultPath string) string {
	if p := spec.Network.HealthPath; p != "" {
		return p
	}
	if p := spec.HealthCheckPath; p != "" {
		return p
	}
	if defaultPath != "" {
		return defaultPath
	}
	return "/health"
}

// workspaceInternalHealthURL returns the in-cluster Kubernetes service URL for
// health probing, bypassing Traefik and ForwardAuth entirely.
// Pattern: http://{serviceName}.{namespace}.svc.cluster.local:8642{healthPath}
// Both service name and namespace equal spec.WorkspaceID by convention.
func workspaceInternalHealthURL(spec WorkspaceSpec, defaultPath string) string {
	ns := spec.Namespace
	if ns == "" {
		ns = spec.WorkspaceID
	}
	svc := spec.WorkspaceID
	if ns == "" || svc == "" {
		return ""
	}
	healthPath := resolveHealthPath(spec, defaultPath)
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:8642%s", svc, ns, healthPath)
}

func convertDeploymentConditions(conditions []appsv1.DeploymentCondition) []metav1.Condition {
	result := make([]metav1.Condition, 0, len(conditions))
	for _, condition := range conditions {
		result = append(result, metav1.Condition{
			Type:    string(condition.Type),
			Status:  metav1.ConditionStatus(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}
	return result
}

func deploymentMessage(deployment appsv1.Deployment) string {
	for _, condition := range deployment.Status.Conditions {
		if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
			if condition.Message != "" {
				return condition.Message
			}
		}
	}
	return ""
}

func podMessage(pod corev1.Pod) string {
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Message != "" {
			return status.State.Waiting.Message
		}
		if status.State.Terminated != nil && status.State.Terminated.Message != "" {
			return status.State.Terminated.Message
		}
	}
	return pod.Status.Message
}

func selectDeployment(items []appsv1.Deployment, releaseName string) *appsv1.Deployment {
	for i := range items {
		if items[i].Name == releaseName {
			return &items[i]
		}
	}
	if len(items) > 0 {
		return &items[0]
	}
	return nil
}

func selectPod(items []corev1.Pod) *corev1.Pod {
	for i := range items {
		if items[i].Status.Phase == corev1.PodRunning {
			return &items[i]
		}
	}
	if len(items) > 0 {
		return &items[0]
	}
	return nil
}
