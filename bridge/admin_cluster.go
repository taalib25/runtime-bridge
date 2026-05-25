package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const managedByLabel = "hermeshq/managed-by=bridge"

// ClusterSummaryResponse is the payload for GET /v1/cluster/summary.
type ClusterSummaryResponse struct {
	ClusterID           string        `json:"clusterId"`
	BridgeHealthy       bool          `json:"bridgeHealthy"`
	KubernetesReachable bool          `json:"kubernetesReachable"`
	NodeCount           int           `json:"nodeCount"`
	InstanceCount       int           `json:"instanceCount"`
	Pods                PodCounts     `json:"pods"`
	Resources           ClusterLoad   `json:"resources"`
	LastSeenAt          time.Time     `json:"lastSeenAt"`
}

type PodCounts struct {
	Running      int `json:"running"`
	Pending      int `json:"pending"`
	Failed       int `json:"failed"`
	CrashLooping int `json:"crashLooping"`
	Total        int `json:"total"`
}

// ClusterLoad is the sum of resource requests across all managed pods.
// Used by the backend to score clusters for instance routing — mirrors how
// the Kubernetes scheduler thinks about capacity (requests, not actual usage).
type ClusterLoad struct {
	ReservedCPUm      int64 `json:"reservedCpuM"`      // millicores
	ReservedMemoryMiB int64 `json:"reservedMemoryMiB"` // mebibytes
}

// ClusterResourcesResponse is the payload for GET /v1/cluster/resources.
type ClusterResourcesResponse struct {
	ClusterID string     `json:"clusterId"`
	Nodes     []NodeInfo `json:"nodes"`
}

type NodeInfo struct {
	Name              string   `json:"name"`
	Ready             bool     `json:"ready"`
	Conditions        []string `json:"conditions"`
	AllocatableCPU    string   `json:"allocatableCpu"`
	AllocatableMemory string   `json:"allocatableMemory"`
	KernelVersion     string   `json:"kernelVersion,omitempty"`
	OSImage           string   `json:"osImage,omitempty"`
	KubeletVersion    string   `json:"kubeletVersion,omitempty"`
}

func (b *Bridge) GetClusterSummary(ctx context.Context) (ClusterSummaryResponse, error) {
	out := ClusterSummaryResponse{
		ClusterID:  b.ClusterName,
		LastSeenAt: time.Now().UTC(),
	}

	// Check Kubernetes reachability via discovery ping.
	_, err := b.KubeClient.Discovery().ServerVersion()
	if err != nil {
		out.BridgeHealthy = true // bridge itself is up
		out.KubernetesReachable = false
		return out, nil
	}
	out.BridgeHealthy = true
	out.KubernetesReachable = true

	// Count nodes.
	nodes, err := b.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, fmt.Errorf("list nodes: %w", err)
	}
	out.NodeCount = len(nodes.Items)

	// Find bridge-managed namespaces.
	nsList, err := b.KubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: managedByLabel,
	})
	if err != nil {
		return out, fmt.Errorf("list managed namespaces: %w", err)
	}
	out.InstanceCount = len(nsList.Items)

	// Build a set of managed namespace names for fast lookup.
	managed := make(map[string]bool, len(nsList.Items))
	for _, ns := range nsList.Items {
		managed[ns.Name] = true
	}

	// List all pods cluster-wide, filter to managed namespaces.
	allPods, err := b.KubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, fmt.Errorf("list pods: %w", err)
	}

	counts := PodCounts{}
	load := ClusterLoad{}
	for i := range allPods.Items {
		pod := &allPods.Items[i]
		if !managed[pod.Namespace] {
			continue
		}
		counts.Total++

		switch pod.Status.Phase {
		case corev1.PodRunning:
			counts.Running++
		case corev1.PodPending:
			counts.Pending++
		case corev1.PodFailed:
			counts.Failed++
		}

		if isCrashLooping(pod) {
			counts.CrashLooping++
		}

		// Sum resource requests across all containers in the pod.
		for _, c := range pod.Spec.Containers {
			if cpu := c.Resources.Requests.Cpu(); cpu != nil {
				load.ReservedCPUm += cpu.MilliValue()
			}
			if mem := c.Resources.Requests.Memory(); mem != nil {
				load.ReservedMemoryMiB += mem.Value() / (1024 * 1024)
			}
		}
	}
	out.Pods = counts
	out.Resources = load
	return out, nil
}

func (b *Bridge) GetClusterResources(ctx context.Context) (ClusterResourcesResponse, error) {
	out := ClusterResourcesResponse{ClusterID: b.ClusterName}

	nodes, err := b.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, fmt.Errorf("list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		ready := false
		var conditions []string
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				ready = cond.Status == corev1.ConditionTrue
			}
			if cond.Status != corev1.ConditionTrue {
				continue
			}
			// Surface active pressure conditions as warnings.
			switch cond.Type {
			case corev1.NodeMemoryPressure, corev1.NodeDiskPressure,
				corev1.NodePIDPressure, corev1.NodeNetworkUnavailable:
				conditions = append(conditions, string(cond.Type))
			}
		}
		if ready {
			conditions = append([]string{"Ready"}, conditions...)
		}

		cpu := ""
		mem := ""
		if q, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
			cpu = q.String()
		}
		if q, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
			mem = q.String()
		}

		out.Nodes = append(out.Nodes, NodeInfo{
			Name:              node.Name,
			Ready:             ready,
			Conditions:        conditions,
			AllocatableCPU:    cpu,
			AllocatableMemory: mem,
			KernelVersion:     node.Status.NodeInfo.KernelVersion,
			OSImage:           node.Status.NodeInfo.OSImage,
			KubeletVersion:    node.Status.NodeInfo.KubeletVersion,
		})
	}
	return out, nil
}

// isCrashLooping returns true if a pod is in CrashLoopBackOff or has restarted
// more than 5 times (catches rapid-cycling containers briefly entering Running).
func isCrashLooping(pod *corev1.Pod) bool {
	if containerWaitingReason(pod) == "CrashLoopBackOff" {
		return true
	}
	return podRestartCount(pod) > 5
}
