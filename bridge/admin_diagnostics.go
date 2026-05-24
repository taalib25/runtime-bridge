package main

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	Recommendation string         `json:"recommendation"`
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
	svcs, err := b.KubeClient.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err == nil && len(svcs.Items) > 0 {
		svc := svcs.Items[0]
		ports := make([]int32, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, p.Port)
		}
		hasEndpoints := false
		ep, epErr := b.KubeClient.CoreV1().Endpoints(ns).Get(ctx, svc.Name, metav1.GetOptions{})
		if epErr == nil {
			for _, sub := range ep.Subsets {
				if len(sub.Addresses) > 0 {
					hasEndpoints = true
					break
				}
			}
		}
		out.Service = &ServiceSummary{
			Name:         svc.Name,
			HasEndpoints: hasEndpoints,
			Ports:        ports,
		}
	}

	// Pod logs — current and previous crash (best-effort, non-fatal)
	pods, _ := b.KubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if pods != nil && len(pods.Items) > 0 {
		pod := selectPod(pods.Items)
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

