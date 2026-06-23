package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const managedByLabel = "hermeshq/managed-by=bridge"

// ClusterSummaryResponse is the payload for GET /v1/cluster/summary.
type ClusterSummaryResponse struct {
	ClusterID           string      `json:"clusterId"`
	BridgeHealthy       bool        `json:"bridgeHealthy"`
	KubernetesReachable bool        `json:"kubernetesReachable"`
	NodeCount           int         `json:"nodeCount"`
	InstanceCount       int         `json:"instanceCount"`
	Pods                PodCounts   `json:"pods"`
	Resources           ClusterLoad `json:"resources"`
	LastSeenAt          time.Time   `json:"lastSeenAt"`
	Maintenance         bool        `json:"maintenance"`
	Draining            bool        `json:"draining"`
	// HeadroomMiB is the estimated free memory available for new instances:
	// allocatable - systemOverhead - reserved. Negative means the cluster is
	// over-committed. Backend should exclude clusters where HeadroomMiB < 512
	// (one instance memory request) from routing.
	HeadroomMiB int64 `json:"headroomMiB"`
	// HeadroomUnknown is true when the node list call needed to compute
	// HeadroomMiB failed (RBAC, timeout, etc). HeadroomMiB is 0 in that case —
	// callers must not read that 0 as "no headroom"; check this flag first.
	HeadroomUnknown bool `json:"headroomUnknown,omitempty"`
	// RuntimeNodesTotal/RuntimeNodesReady and the Prepuller* fields are all read
	// directly off the hermes-runtime-image-prepuller DaemonSet's own status — not a
	// separate count — so "how many runtime nodes are there" and "how many have the
	// image cached" can never drift apart from what the DaemonSet itself believes.
	// All zero/empty when the DaemonSet hasn't been deployed yet (e.g. an older
	// cluster, or a deploy that predates Step 4) — that's a real "not warmed" signal.
	// If the lookup itself failed instead (RBAC, timeout), that is NOT the same
	// thing — see RuntimeImageStatusUnknown.
	RuntimeNodesTotal     int    `json:"runtimeNodesTotal"`
	RuntimeNodesReady     int    `json:"runtimeNodesReady"`
	PrepullerDesired      int    `json:"prepullerDesired"`
	PrepullerReady        int    `json:"prepullerReady"`
	PrepullerUnavailable  int    `json:"prepullerUnavailable"`
	RuntimeImageReference string `json:"runtimeImageReference,omitempty"`
	// RuntimeImageWarmed is true only when every node the pre-puller targets has it
	// cached (prepullerReady == prepullerDesired, and desired > 0). Backends should
	// prefer routing creates to clusters where this is true and avoid placing the
	// FIRST instance on a cluster where it's false — see docs on warm-node validation.
	RuntimeImageWarmed bool `json:"runtimeImageWarmed"`
	// RuntimeImageStatusUnknown is true when the DaemonSet lookup itself failed
	// (RBAC, timeout) rather than returning a real "not deployed" or "not warmed"
	// state. This previously could not be distinguished from a genuine cold cluster —
	// the bridge's own ClusterRole missing the daemonsets grant made every cluster
	// look permanently un-warmed. See deploy/rbac.yaml.
	RuntimeImageStatusUnknown bool `json:"runtimeImageStatusUnknown,omitempty"`
}

// MaintenanceModeResponse is the payload for GET/PUT /v1/cluster/maintenance.
type MaintenanceModeResponse struct {
	ClusterID   string    `json:"clusterId"`
	Maintenance bool      `json:"maintenance"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	out.Maintenance = b.maintenance.Load()
	out.Draining = b.draining.Load()

	// Compute headroom: sum allocatable memory across ready nodes, subtract
	// systemOverheadMiB and the reserved request total.
	if allNodes, nodeErr := b.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); nodeErr == nil {
		nodes = allNodes
		var allocatableMiB int64
		for _, node := range nodes.Items {
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					if q, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
						allocatableMiB += q.Value() / (1024 * 1024)
					}
				}
			}
		}
		out.HeadroomMiB = allocatableMiB - systemOverheadMiB - out.Resources.ReservedMemoryMiB
	} else {
		// Don't let HeadroomMiB's zero value masquerade as "no headroom" — that
		// would make a healthy cluster look full to the backend's routing filter.
		out.HeadroomUnknown = true
		b.Logger.Printf("[GetClusterSummary] Warning: re-list nodes for headroom failed: %v", nodeErr)
	}

	if ds, dsErr := b.KubeClient.AppsV1().DaemonSets("hermes-system").Get(ctx, "hermes-runtime-image-prepuller", metav1.GetOptions{}); dsErr == nil {
		out.RuntimeNodesTotal = int(ds.Status.DesiredNumberScheduled)
		out.RuntimeNodesReady = int(ds.Status.NumberReady)
		out.PrepullerDesired = int(ds.Status.DesiredNumberScheduled)
		out.PrepullerReady = int(ds.Status.NumberReady)
		out.PrepullerUnavailable = int(ds.Status.NumberUnavailable)
		if len(ds.Spec.Template.Spec.Containers) > 0 {
			out.RuntimeImageReference = ds.Spec.Template.Spec.Containers[0].Image
		}
		out.RuntimeImageWarmed = out.PrepullerDesired > 0 && out.PrepullerReady == out.PrepullerDesired
	} else if !apierrors.IsNotFound(dsErr) {
		// NotFound is a real "DaemonSet isn't deployed" signal — leave the zero
		// values as-is. Anything else (RBAC Forbidden, timeout) is "couldn't check",
		// which previously looked identical to "checked, found nothing warmed".
		out.RuntimeImageStatusUnknown = true
		b.Logger.Printf("[GetClusterSummary] Warning: get prepuller DaemonSet failed: %v", dsErr)
	}

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

// DrainCluster enables maintenance mode then deletes every managed instance.
// purge=true permanently destroys namespace + PVC data (same as DELETE with purge).
// Returns a summary of deleted and failed instance IDs.
func (b *Bridge) DrainCluster(ctx context.Context, purge bool) error {
	b.maintenance.Store(true)

	if b.KubeClient == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	nsList, err := b.KubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: managedByLabel,
	})
	if err != nil {
		return fmt.Errorf("list managed namespaces: %w", err)
	}

	var failed []string
	for _, ns := range nsList.Items {
		instanceID := ns.Name
		// Skip already-tombstoned namespaces unless purge=true.
		// Tombstoned instances have hermes.io/deleted-at set; their Helm release is
		// already gone and the namespace is kept intentionally for PVC/history.
		// With purge=true we want to destroy them, so we fall through.
		if !purge && ns.Annotations["hermes.io/deleted-at"] != "" {
			b.Logger.Printf("[DrainCluster] Skipping tombstoned instance %s", instanceID)
			continue
		}
		if err := b.DeleteInstance(ctx, instanceID, purge); err != nil {
			b.Logger.Printf("[DrainCluster] Failed to delete instance %s: %v", instanceID, err)
			failed = append(failed, instanceID)
		} else {
			b.Logger.Printf("[DrainCluster] Deleted instance %s", instanceID)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("drain completed with %d failure(s): %v", len(failed), failed)
	}
	b.Logger.Printf("[DrainCluster] Drained %d instances (purge=%v)", len(nsList.Items), purge)
	return nil
}

// isCrashLooping returns true if a pod is in CrashLoopBackOff or has restarted
// more than 5 times (catches rapid-cycling containers briefly entering Running).
func isCrashLooping(pod *corev1.Pod) bool {
	if containerWaitingReason(pod) == "CrashLoopBackOff" {
		return true
	}
	return podRestartCount(pod) > 5
}
