package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// DiagnosticsResponse is the payload for GET /v1/instances/{id}/diagnostics.
type DiagnosticsResponse struct {
	InstanceID    string         `json:"instanceId"`
	Namespace      string         `json:"namespace"`
	ReleaseName    string         `json:"releaseName"`
	Status         InstanceStatus `json:"status"`
	PVC            *PVCSummary    `json:"pvc,omitempty"`
	Service        *ServiceSummary `json:"service,omitempty"`
	Logs           LogsSummary    `json:"logs"`
	Events         []InstanceEvent `json:"events"`
	Timeline       ProvisionTimeline `json:"timeline"`
	Recommendation string         `json:"recommendation"`
}

// ProvisionTimeline answers "where did the time go" for a single provision — image
// pull, container start, app boot, or Service/Endpoints propagation — instead of one
// opaque "still starting" status. Every *At field is nil when the underlying signal
// hasn't happened yet (or never will, e.g. ImagePullStartedAt stays nil for an
// already-cached image) — nil is "hasn't happened", not "unknown"/an error.
type ProvisionTimeline struct {
	PodCreatedAt            *time.Time `json:"podCreatedAt,omitempty"`
	PodScheduledAt          *time.Time `json:"podScheduledAt,omitempty"`
	ImagePullStartedAt      *time.Time `json:"imagePullStartedAt,omitempty"`
	ImagePulledAt           *time.Time `json:"imagePulledAt,omitempty"`
	ContainerCreatedAt      *time.Time `json:"containerCreatedAt,omitempty"`
	ContainerStartedAt      *time.Time `json:"containerStartedAt,omitempty"`
	PodReadyAt              *time.Time `json:"podReadyAt,omitempty"`
	ServiceEndpointsReadyAt *time.Time `json:"serviceEndpointsReadyAt,omitempty"`
	// FirstAPIStatusSuccessAt is read off the Pod's own Ready condition transition time:
	// the readinessProbe hits the same /api/status path checkInstanceHealth does (see
	// charts/hermes-agent/values.yaml probes.readiness), so "Pod went Ready" and "the
	// app first answered /api/status with 2xx" are the same event, not two separate
	// signals that can silently disagree.
	FirstAPIStatusSuccessAt *time.Time `json:"firstApiStatusSuccessAt,omitempty"`

	ScheduleSeconds       *float64 `json:"scheduleSeconds,omitempty"`
	ImagePullSeconds      *float64 `json:"imagePullSeconds,omitempty"`
	ContainerStartSeconds *float64 `json:"containerStartSeconds,omitempty"`
	AppBootSeconds        *float64 `json:"appBootSeconds,omitempty"`
	ServiceReadySeconds   *float64 `json:"serviceReadySeconds,omitempty"`
	TotalProvisionSeconds *float64 `json:"totalProvisionSeconds,omitempty"`
}

type PVCSummary struct {
	Name     string `json:"name"`
	Phase    string `json:"phase"`
	Capacity string `json:"capacity,omitempty"`
}

type ServiceSummary struct {
	Name         string  `json:"name"`
	HasEndpoints bool    `json:"hasEndpoints"`
	Ports        []int32 `json:"ports"`
	// ReadyCount/NotReadyCount/Source make the diagnosis self-evident instead of a bare
	// bool: e.g. "0 ready, 0 not-ready, source=endpointslice" vs "source=error" (a real
	// failure to check, NOT the same thing as "checked and found nothing").
	ReadyCount    int    `json:"readyCount"`
	NotReadyCount int    `json:"notReadyCount"`
	Source        string `json:"source"`
}

// endpointReadiness inspects EndpointSlices first (the modern API; what kube-proxy and
// most CNIs actually consume) and falls back to the legacy Endpoints object only if no
// EndpointSlice exists. Returns the source used and ready/not-ready address counts.
// Critically, this surfaces lookup ERRORS distinctly from "no endpoints found" — a
// previous version silently treated an RBAC 403 on Endpoints the same as "not ready",
// which made the bridge's own ClusterRole missing the `endpoints` resource look like a
// real pod health problem. See deploy/rbac.yaml.
func (b *Bridge) endpointReadiness(ctx context.Context, ns, svcName string) (source string, ready, notReady int, err error) {
	slices, sliceErr := b.KubeClient.DiscoveryV1().EndpointSlices(ns).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", svcName),
	})
	if sliceErr == nil && len(slices.Items) > 0 {
		for _, slice := range slices.Items {
			for _, ep := range slice.Endpoints {
				if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
					ready += len(ep.Addresses)
				} else {
					notReady += len(ep.Addresses)
				}
			}
		}
		return "endpointslice", ready, notReady, nil
	}

	ep, epErr := b.KubeClient.CoreV1().Endpoints(ns).Get(ctx, svcName, metav1.GetOptions{})
	if epErr != nil {
		// Both lookups failed — this is a genuine error (RBAC, network, wrong name),
		// not "the pod isn't ready yet". Callers must not coerce this into hasEndpoints=false
		// without surfacing it.
		if sliceErr != nil {
			return "error", 0, 0, fmt.Errorf("list endpointslices: %w; get endpoints: %v", sliceErr, epErr)
		}
		return "error", 0, 0, fmt.Errorf("get endpoints: %w", epErr)
	}
	for _, sub := range ep.Subsets {
		ready += len(sub.Addresses)
		notReady += len(sub.NotReadyAddresses)
	}
	return "endpoints", ready, notReady, nil
}

type LogsSummary struct {
	Current  string `json:"current"`
	Previous string `json:"previous,omitempty"`
}

// LogsResponse is the payload for GET /v1/instances/{id}/logs.
type LogsResponse struct {
	InstanceID string `json:"instanceId"`
	PodName     string `json:"podName"`
	Container   string `json:"container"`
	Tail        int64  `json:"tail"`
	Previous    bool   `json:"previous"`
	Logs        string `json:"logs"`
}

func (b *Bridge) GetInstanceDiagnostics(ctx context.Context, instanceID string) (DiagnosticsResponse, error) {
	status, err := b.GetInstanceStatus(ctx, instanceID)
	if err != nil {
		return DiagnosticsResponse{}, err
	}
	ns := status.Namespace
	releaseName := status.ReleaseName

	out := DiagnosticsResponse{
		InstanceID: instanceID,
		Namespace:   ns,
		ReleaseName: releaseName,
		Status:      status,
	}

	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()

	// PVC summary
	pvcs, err := b.KubeClient.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil && len(pvcs.Items) > 0 {
		pvc := pvcs.Items[0]
		cap := ""
		if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			cap = storage.String()
		}
		out.PVC = &PVCSummary{
			Name:     pvc.Name,
			Phase:    string(pvc.Status.Phase),
			Capacity: cap,
		}
	}

	// Service + endpoints summary
	var serviceEndpointsReadyAt *time.Time
	svcs, err := b.KubeClient.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil && len(svcs.Items) > 0 {
		svc := svcs.Items[0]
		ports := make([]int32, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, p.Port)
		}
		source, ready, notReady, readinessErr := b.endpointReadiness(ctx, ns, svc.Name)
		if readinessErr != nil {
			// Surface the failure in the log — a 403 here means the bridge's own
			// ClusterRole is missing the endpoints/endpointslices grant (deploy/rbac.yaml),
			// which is a platform bug, not "the instance isn't ready".
			b.Logger.Printf("[GetInstanceDiagnostics] Warning: endpoint readiness check failed for %s/%s: %v", ns, svc.Name, readinessErr)
		}
		out.Service = &ServiceSummary{
			Name:          svc.Name,
			HasEndpoints:  ready > 0,
			Ports:         ports,
			ReadyCount:    ready,
			NotReadyCount: notReady,
			Source:        source,
		}
		serviceEndpointsReadyAt = endpointsLastChangeTriggerTime(b.endpointsAnnotations(ctx, ns, svc.Name))
	}

	// Pod logs — current and previous crash (best-effort, non-fatal)
	var pod *corev1.Pod
	pods, _ := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if pods != nil && len(pods.Items) > 0 {
		pod = selectPod(pods.Items)
		if pod == nil {
			pod = &pods.Items[0]
		}
		container := ""
		if len(pod.Spec.Containers) > 0 {
			container = pod.Spec.Containers[0].Name
		}
		tail := int64(200)
		out.Logs.Current = b.fetchPodLogs(ctx, ns, pod.Name, container, tail, false)
		out.Logs.Previous = b.fetchPodLogs(ctx, ns, pod.Name, container, tail, true)
	}

	// Events
	events, _ := b.GetInstanceEvents(ctx, instanceID)
	if events == nil {
		events = []InstanceEvent{}
	}
	out.Events = events

	out.Timeline = buildProvisionTimeline(pod, events, serviceEndpointsReadyAt)
	b.Logger.Printf(
		"[GetInstanceDiagnostics] %s timeline: scheduleSeconds=%s imagePullSeconds=%s containerStartSeconds=%s appBootSeconds=%s serviceReadySeconds=%s totalProvisionSeconds=%s",
		instanceID,
		formatSecondsForLog(out.Timeline.ScheduleSeconds),
		formatSecondsForLog(out.Timeline.ImagePullSeconds),
		formatSecondsForLog(out.Timeline.ContainerStartSeconds),
		formatSecondsForLog(out.Timeline.AppBootSeconds),
		formatSecondsForLog(out.Timeline.ServiceReadySeconds),
		formatSecondsForLog(out.Timeline.TotalProvisionSeconds),
	)

	pvcBound := out.PVC != nil && strings.EqualFold(out.PVC.Phase, "Bound")
	svcHasEndpoints := out.Service != nil && out.Service.HasEndpoints
	out.Recommendation = recommendAction(status, pvcBound, svcHasEndpoints)

	return out, nil
}

// GetInstanceLogs fetches pod logs for the given workspace.
func (b *Bridge) GetInstanceLogs(ctx context.Context, instanceID, container string, tail int64, previous bool) (LogsResponse, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return LogsResponse{}, err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	pods, err := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return LogsResponse{}, fmt.Errorf("list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return LogsResponse{}, fmt.Errorf("no pods found for instance %s", instanceID)
	}

	pod := selectPod(pods.Items)
	if pod == nil {
		pod = &pods.Items[0]
	}
	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}

	logs := b.fetchPodLogs(ctx, ns, pod.Name, container, tail, previous)
	return LogsResponse{
		InstanceID: instanceID,
		PodName:     pod.Name,
		Container:   container,
		Tail:        tail,
		Previous:    previous,
		Logs:        logs,
	}, nil
}

// fetchPodLogs reads pod logs into a string. Returns empty string on any error
// so callers can treat it as best-effort.
func (b *Bridge) fetchPodLogs(ctx context.Context, ns, podName, container string, tail int64, previous bool) string {
	opts := &corev1.PodLogOptions{
		Container: container,
		TailLines: &tail,
		Previous:  previous,
	}
	req := b.KubeClient.CoreV1().Pods(ns).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return ""
	}
	defer stream.Close()

	// Guard against unexpectedly large log blobs (safety cap at 512KB).
	lr := io.LimitReader(stream, 512*1024)
	raw, err := io.ReadAll(lr)
	if err != nil {
		return ""
	}
	return string(raw)
}

// recommendAction derives a human-readable admin recommendation from instance signals.
func recommendAction(status InstanceStatus, pvcBound, serviceHasEndpoints bool) string {
	switch status.WaitingReason {
	case "CrashLoopBackOff":
		if status.RestartCount > 5 {
			return "Container crashing repeatedly — check logs for startup errors or missing env vars"
		}
		return "Container crashed — check current and previous logs for the root cause"
	case "OOMKilled":
		return "Pod killed by OOM — increase memory limits via PUT /v1/instances/{id}"
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return "Cannot pull image — check image name and tag; verify pull credentials"
	case "CreateContainerConfigError":
		return "Container config error — a referenced Secret key or ConfigMap may be missing"
	case "CreateContainerError", "RunContainerError":
		return "Container failed to start — check events and node conditions"
	case "Unschedulable":
		return "No node has available capacity — cluster may be full or node selectors don't match"
	}

	if strings.HasPrefix(status.WaitingReason, "Init:") {
		return "Init container stuck — check init container logs and events"
	}

	if !pvcBound {
		return "PVC not bound — storage provisioner may be stuck; check events"
	}

	switch status.Phase {
	case "ready":
		if !serviceHasEndpoints {
			return "Instance is healthy but service has no endpoints — pod may not have completed readiness probe"
		}
		return "Instance is healthy — no action required"
	case "starting":
		return "Instance is starting up — wait for health check to pass; check logs if this persists"
	case "creating":
		return "Instance is being provisioned — wait a moment"
	case "failed":
		return "Pod failed — check events; use POST /v1/instances/{id}/repair to attempt recovery"
	case "error":
		return "Instance is in error state — check logs and events; use POST /v1/instances/{id}/repair"
	case "deleted":
		return "Instance has been deleted"
	}

	return "Check logs and events for more details; use POST /v1/instances/{id}/repair if degraded"
}

// endpointsAnnotations fetches the Endpoints object's annotations purely to read
// "endpoints.kubernetes.io/last-change-trigger-time" — the timestamp Kubernetes itself
// stamps whenever the object's ready addresses last changed. Best-effort: returns nil
// on any error rather than failing the whole diagnostics call over a timing nice-to-have.
func (b *Bridge) endpointsAnnotations(ctx context.Context, ns, svcName string) map[string]string {
	ep, err := b.KubeClient.CoreV1().Endpoints(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return ep.Annotations
}

func endpointsLastChangeTriggerTime(annotations map[string]string) *time.Time {
	raw, ok := annotations["endpoints.kubernetes.io/last-change-trigger-time"]
	if !ok || raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

// eventTimeByReason finds the earliest FirstTime (if earliest=true) or latest LastTime
// across events matching reason, optionally restricted to events whose Message mentions
// containerName (multi-container pods emit one Pulling/Pulled/Created/Started event per
// container, sharing the same Reason — without the message filter we'd conflate the init
// containers' timings with the main runtime container's).
func eventTimeByReason(events []InstanceEvent, reason, containerName string, earliest bool) *time.Time {
	var best *time.Time
	for i := range events {
		e := &events[i]
		if e.Reason != reason {
			continue
		}
		if containerName != "" && !strings.Contains(e.Message, containerName) {
			continue
		}
		candidate := e.LastTime
		if earliest {
			candidate = e.FirstTime
		}
		if candidate.IsZero() {
			continue
		}
		if best == nil || (earliest && candidate.Before(*best)) || (!earliest && candidate.After(*best)) {
			best = &candidate
		}
	}
	return best
}

func formatSecondsForLog(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.1fs", *v)
}

func secondsBetween(start, end *time.Time) *float64 {
	if start == nil || end == nil {
		return nil
	}
	d := end.Sub(*start).Seconds()
	if d < 0 {
		return nil
	}
	return &d
}

func podConditionTime(pod *corev1.Pod, conditionType corev1.PodConditionType) *time.Time {
	if pod == nil {
		return nil
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == conditionType && c.Status == corev1.ConditionTrue && !c.LastTransitionTime.IsZero() {
			t := c.LastTransitionTime.Time
			return &t
		}
	}
	return nil
}

// buildProvisionTimeline is pure (no I/O) so it's directly unit-testable: feed it a pod
// + event list + an optional endpoints-readiness timestamp and get back exactly the
// breakdown described in the startup-instrumentation spec, with nil for any phase that
// hasn't happened (or wasn't observable) rather than a misleading zero duration.
func buildProvisionTimeline(pod *corev1.Pod, events []InstanceEvent, serviceEndpointsReadyAt *time.Time) ProvisionTimeline {
	tl := ProvisionTimeline{ServiceEndpointsReadyAt: serviceEndpointsReadyAt}
	if pod == nil {
		return tl
	}

	created := pod.CreationTimestamp.Time
	if !created.IsZero() {
		tl.PodCreatedAt = &created
	}

	mainContainer := ""
	if len(pod.Spec.Containers) > 0 {
		mainContainer = pod.Spec.Containers[0].Name
	}

	tl.PodScheduledAt = podConditionTime(pod, corev1.PodScheduled)
	if tl.PodScheduledAt == nil {
		tl.PodScheduledAt = eventTimeByReason(events, "Scheduled", "", true)
	}

	// Pulling/Pulled fire once per container (init containers included) — use the
	// overall earliest Pulling and latest Pulled to represent the pod's full image-pull
	// phase, since a slow init-container pull delays the main container just as much.
	tl.ImagePullStartedAt = eventTimeByReason(events, "Pulling", "", true)
	tl.ImagePulledAt = eventTimeByReason(events, "Pulled", "", false)

	tl.ContainerCreatedAt = eventTimeByReason(events, "Created", mainContainer, true)
	tl.ContainerStartedAt = eventTimeByReason(events, "Started", mainContainer, true)

	// The Pod's Ready condition is driven by the readinessProbe, which hits the same
	// /api/status path checkInstanceHealth does — so this is also the first time the
	// app answered a real request, not just "the process is running".
	tl.PodReadyAt = podConditionTime(pod, corev1.PodReady)
	tl.FirstAPIStatusSuccessAt = tl.PodReadyAt

	tl.ScheduleSeconds = secondsBetween(tl.PodCreatedAt, tl.PodScheduledAt)
	tl.ImagePullSeconds = secondsBetween(tl.ImagePullStartedAt, tl.ImagePulledAt)
	tl.ContainerStartSeconds = secondsBetween(tl.ContainerCreatedAt, tl.ContainerStartedAt)
	tl.AppBootSeconds = secondsBetween(tl.ContainerStartedAt, tl.PodReadyAt)
	tl.ServiceReadySeconds = secondsBetween(tl.PodReadyAt, tl.ServiceEndpointsReadyAt)
	tl.TotalProvisionSeconds = secondsBetween(tl.PodCreatedAt, tl.PodReadyAt)

	return tl
}

