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
	helmrelease "helm.sh/helm/v3/pkg/release"
)

func (b *Bridge) GetInstanceStatus(ctx context.Context, instanceID string) (InstanceStatus, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return InstanceStatus{}, err
	}
	// Release exists but has been soft-deleted (--keep-history). Return a
	// minimal deleted status so the backend can stop polling and mark it done.
	if rel.Info != nil && rel.Info.Status == helmrelease.StatusUninstalled {
		spec, _ := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
		return InstanceStatus{
			InstanceID: instanceID,
			ClusterID:   b.ClusterName,
			ReleaseName: rel.Name,
			Namespace:   rel.Namespace,
			Phase:       "deleted",
			CreatedAt:   rel.Info.FirstDeployed.Time.UTC(),
			LastCheckedAt: time.Now().UTC(),
			Spec:        spec,
		}, nil
	}
	spec, err := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return InstanceStatus{}, err
	}
	return b.collectInstanceStatus(ctx, spec, rel.Name, rel.Info.FirstDeployed.Time)
}

func (b *Bridge) collectInstanceStatus(ctx context.Context, spec InstanceSpec, releaseName string, createdAt time.Time) (InstanceStatus, error) {
	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	namespace := b.instanceNamespace(spec)

	deployments, err := b.KubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return InstanceStatus{}, fmt.Errorf("list deployments: %w", err)
	}
	pods, err := b.KubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return InstanceStatus{}, fmt.Errorf("list pods: %w", err)
	}

	status := InstanceStatus{
		InstanceID:   spec.InstanceID,
		ClusterID:     spec.ClusterID,
		ReleaseName:   releaseName,
		Namespace:     namespace,
		URL:           instanceURL(spec),
		DashboardURL:  dashboardURL(spec),
		CreatedAt:     createdAt.UTC(),
		LastCheckedAt: time.Now().UTC(),
		Spec:          spec,
		// Echo the effective label contract so the backend can verify round-trip.
		SelectorLabels:    deriveSelectorLabels(releaseName),
		CommonLabels:      spec.CommonLabels,
		CommonAnnotations: spec.CommonAnnotations,
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

	healthy, code, healthErr := b.checkInstanceHealth(ctx, spec)
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
	// Pod not yet scheduled — cluster may be full or all nodes are cordoned.
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
			return "Unschedulable"
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

func (b *Bridge) checkInstanceHealth(ctx context.Context, spec InstanceSpec) (bool, int, error) {
	// Probe the internal Kubernetes service directly so health checks bypass
	// Traefik ForwardAuth — the pod is healthy even before the auth endpoint exists.
	url := b.instanceInternalHealthURL(spec, b.Config.HealthPath)
	if url == "" {
		return false, 0, fmt.Errorf("instance URL is not configured")
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

func derivePhase(status InstanceStatus, healthErr error) string {
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
	case "Unschedulable":
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

// dashboardURL is the tenant's access URL. The main host serves the Hermes web
// dashboard (port 9119) directly, so it is just the instance host.
func dashboardURL(spec InstanceSpec) string {
	h := spec.Network.host()
	if h == "" {
		return ""
	}
	scheme := spec.Network.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + h
}

func instanceURL(spec InstanceSpec) string {
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

func resolveHealthPath(spec InstanceSpec, defaultPath string) string {
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
// Pattern: http://{serviceName}.{namespace}.svc.cluster.local:{port}{healthPath}
// The service name equals the Helm release name (releaseName = ReleasePrefix + WorkspaceID)
// because buildValues sets fullnameOverride to that value.
func (b *Bridge) instanceInternalHealthURL(spec InstanceSpec, defaultPath string) string {
	ns := spec.Namespace
	if ns == "" {
		ns = spec.InstanceID
	}
	svc := b.releaseName(spec.InstanceID)
	if ns == "" || svc == "" {
		return ""
	}
	port := spec.RuntimePort
	if port == 0 {
		port = 8787 // safe fallback for releases created before RuntimePort was introduced
	}
	healthPath := resolveHealthPath(spec, defaultPath)
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", svc, ns, port, healthPath)
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

// selectPod picks the most representative pod for status reporting. During a
// rolling update there are briefly two pods (old terminating + new starting);
// prefer the newest Running pod so status reflects the incoming revision rather
// than the doomed one. Falls back to the newest pod of any phase.
func selectPod(items []corev1.Pod) *corev1.Pod {
	var running *corev1.Pod
	var newest *corev1.Pod
	for i := range items {
		p := &items[i]
		if newest == nil || p.CreationTimestamp.After(newest.CreationTimestamp.Time) {
			newest = p
		}
		if p.Status.Phase == corev1.PodRunning {
			if running == nil || p.CreationTimestamp.After(running.CreationTimestamp.Time) {
				running = p
			}
		}
	}
	if running != nil {
		return running
	}
	return newest
}
