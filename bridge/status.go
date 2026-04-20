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
		CreatedAt:     createdAt.UTC(),
		LastCheckedAt: time.Now().UTC(),
		Spec:          spec,
	}

	if len(deployments.Items) > 0 {
		deployment := selectDeployment(deployments.Items, releaseName)
		status.Replicas = deployment.Status.Replicas
		status.ReadyReplicas = deployment.Status.ReadyReplicas
		status.Conditions = deploymentConditions(convertDeploymentConditions(deployment.Status.Conditions))
		status.Message = deploymentMessage(deployment)
	}

	if len(pods.Items) > 0 {
		pod := selectPod(pods.Items)
		status.PodPhase = string(pod.Status.Phase)
		if message := podMessage(pod); message != "" {
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

func (b *Bridge) checkWorkspaceHealth(ctx context.Context, spec WorkspaceSpec) (bool, int, error) {
	url := workspaceHealthURL(spec, b.Config.HealthPath)
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
	if strings.EqualFold(status.PodPhase, string(corev1.PodFailed)) {
		return "failed"
	}
	if status.ReadyReplicas > 0 && status.Healthy && healthErr == nil {
		return "running"
	}
	if status.Replicas == 0 && status.ReadyReplicas == 0 {
		return "creating"
	}
	if strings.EqualFold(status.PodPhase, string(corev1.PodPending)) {
		return "creating"
	}
	if healthErr != nil && status.ReadyReplicas == 0 {
		return "failed"
	}
	if status.ReadyReplicas < status.Replicas {
		return "creating"
	}
	if !status.Healthy {
		return "failed"
	}
	return "creating"
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

func workspaceHealthURL(spec WorkspaceSpec, defaultPath string) string {
	base := workspaceURL(spec)
	if base == "" {
		return ""
	}
	healthPath := spec.Network.HealthPath
	if healthPath == "" {
		healthPath = spec.HealthCheckPath
	}
	if healthPath == "" {
		healthPath = defaultPath
	}
	if healthPath == "" {
		healthPath = "/healthz"
	}
	if strings.HasPrefix(healthPath, "/") {
		return strings.TrimRight(base, "/") + healthPath
	}
	return strings.TrimRight(base, "/") + "/" + healthPath
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

func selectDeployment(items []appsv1.Deployment, releaseName string) appsv1.Deployment {
	for _, item := range items {
		if item.Name == releaseName {
			return item
		}
	}
	return items[0]
}

func selectPod(items []corev1.Pod) corev1.Pod {
	for _, item := range items {
		if item.Status.Phase == corev1.PodRunning {
			return item
		}
	}
	return items[0]
}
