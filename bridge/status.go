package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	helmrelease "helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
			InstanceID:    instanceID,
			ClusterID:     b.ClusterName,
			ReleaseName:   rel.Name,
			Namespace:     rel.Namespace,
			Phase:         "deleted",
			CreatedAt:     rel.Info.FirstDeployed.Time.UTC(),
			LastCheckedAt: time.Now().UTC(),
			Spec:          spec,
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
		InstanceID:    spec.InstanceID,
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

	var pod *corev1.Pod
	if p := selectPod(pods.Items); p != nil {
		pod = p
		status.PodPhase = string(pod.Status.Phase)
		status.WaitingReason = containerWaitingReason(pod)
		status.RestartCount = podRestartCount(pod)
		status.NodeName = pod.Spec.NodeName
		status.PodName = pod.Name
		if len(pod.Status.ContainerStatuses) > 0 {
			status.ImageID = pod.Status.ContainerStatuses[0].ImageID
		}
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

	// Best-effort: events power DetailedPhase + LatestEvent*. A failure here shouldn't
	// fail the whole status call — it just means we fall back to the coarse Phase.
	events, eventsErr := b.GetInstanceEvents(ctx, spec.InstanceID)
	if eventsErr != nil {
		b.Logger.Printf("[collectInstanceStatus] Warning: failed to fetch events for %s: %v", spec.InstanceID, eventsErr)
	}
	status.DetailedPhase = deriveDetailedPhase(status.Phase, pod, events)
	if latest := latestEvent(events); latest != nil {
		status.LatestEventReason = latest.Reason
		status.LatestEventMessage = latest.Message
		t := latest.LastTime
		status.LatestEventTimestamp = &t
	}

	return status, nil
}

// deriveDetailedPhase refines the coarse Phase into the granular progression spelled
// out in the startup-instrumentation spec, reusing the same event/condition parsing
// buildProvisionTimeline already does — so "what phase is it in" and "how long did each
// phase take" can never disagree, they're computed from the same inputs.
func deriveDetailedPhase(coarsePhase string, pod *corev1.Pod, events []InstanceEvent) string {
	switch coarsePhase {
	case "failed", "error", "deleted", "deleting":
		return coarsePhase
	case "ready":
		return "runtime_healthy"
	}

	tl := buildProvisionTimeline(pod, events, nil)
	switch {
	case tl.PodReadyAt != nil:
		return "app_starting"
	case tl.ContainerStartedAt != nil:
		return "container_started"
	case tl.ContainerCreatedAt != nil:
		return "container_creating"
	case tl.ImagePulledAt != nil:
		return "image_pulled"
	case tl.ImagePullStartedAt != nil:
		return "pulling_image"
	case tl.PodScheduledAt != nil:
		return "pod_scheduled"
	default:
		return "pod_pending"
	}
}

// latestEvent returns the event with the most recent LastTime, or nil if events is empty.
func latestEvent(events []InstanceEvent) *InstanceEvent {
	var latest *InstanceEvent
	for i := range events {
		if latest == nil || events[i].LastTime.After(latest.LastTime) {
			latest = &events[i]
		}
	}
	return latest
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

// GetInstanceEvents returns the 50 most recent Kubernetes events for resources
// belonging to this workspace release (deployment, pods, PVCs, service).
func (b *Bridge) GetInstanceEvents(ctx context.Context, instanceID string) ([]InstanceEvent, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	// Build an allowlist of resource names owned by this release.
	relevant := map[string]bool{releaseName: true}
	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()

	// Each List below is best-effort: a failure here narrows the allowlist (we
	// see fewer events than exist), not an error worth failing the whole call
	// for. But it must be logged — silently swallowing it looks identical to
	// "this release legitimately has no pods/pvcs/replicasets", which it isn't.
	pods, err := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range pods.Items {
			relevant[pods.Items[i].Name] = true
		}
	} else {
		b.Logger.Printf("[GetInstanceEvents] Warning: list pods for %s/%s failed: %v", ns, releaseName, err)
	}
	pvcs, err := b.KubeClient.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range pvcs.Items {
			relevant[pvcs.Items[i].Name] = true
		}
	} else {
		b.Logger.Printf("[GetInstanceEvents] Warning: list pvcs for %s/%s failed: %v", ns, releaseName, err)
	}
	replicasets, err := b.KubeClient.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for i := range replicasets.Items {
			relevant[replicasets.Items[i].Name] = true
		}
	} else {
		b.Logger.Printf("[GetInstanceEvents] Warning: list replicasets for %s/%s failed: %v", ns, releaseName, err)
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

// waitForInstanceHealth polls the internal health probe until it returns 2xx or the context deadline passes.
func (b *Bridge) waitForInstanceHealth(ctx context.Context, instanceID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return err
	}
	spec, err := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
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
			return fmt.Errorf("instance %s did not become healthy within %s", instanceID, timeout)
		case <-ticker.C:
			if healthy, _, _ := b.checkInstanceHealth(ctx, spec); healthy {
				return nil
			}
		}
	}
}
