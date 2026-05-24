package main

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ResourcesResponse is the payload for GET /v1/instances/{id}/resources.
type ResourcesResponse struct {
	InstanceID string             `json:"instanceId"`
	Namespace   string             `json:"namespace"`
	Deployments []DeploymentInfo   `json:"deployments"`
	Pods        []PodInfo          `json:"pods"`
	Services    []ServiceInfo      `json:"services"`
	PVCs        []PVCInfo          `json:"pvcs"`
	Secrets     []SecretMeta       `json:"secrets"`
	Ingress     []IngressInfo      `json:"ingress"`
}

type DeploymentInfo struct {
	Name              string `json:"name"`
	DesiredReplicas   int32  `json:"desiredReplicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
}

type PodInfo struct {
	Name         string `json:"name"`
	Phase        string `json:"phase"`
	Ready        bool   `json:"ready"`
	NodeName     string `json:"nodeName,omitempty"`
	RestartCount int32  `json:"restartCount"`
	WaitingReason string `json:"waitingReason,omitempty"`
}

type ServiceInfo struct {
	Name         string  `json:"name"`
	ClusterIP    string  `json:"clusterIP,omitempty"`
	HasEndpoints bool    `json:"hasEndpoints"`
	Ports        []int32 `json:"ports"`
}

type PVCInfo struct {
	Name     string `json:"name"`
	Phase    string `json:"phase"`
	Capacity string `json:"capacity,omitempty"`
	Class    string `json:"storageClass,omitempty"`
}

// SecretMeta lists secret key names only — values are never returned.
type SecretMeta struct {
	Name      string   `json:"name"`
	KeyNames  []string `json:"keyNames"`
}

type IngressInfo struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Hosts []string `json:"hosts,omitempty"`
}

func (b *Bridge) GetInstanceResources(ctx context.Context, instanceID string) (ResourcesResponse, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return ResourcesResponse{}, err
	}
	ns := rel.Namespace
	releaseName := rel.Name

	selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()
	listOpts := metav1.ListOptions{LabelSelector: selector}
	allOpts := metav1.ListOptions{}

	out := ResourcesResponse{
		InstanceID: instanceID,
		Namespace:   ns,
	}

	// Deployments
	deps, err := b.KubeClient.AppsV1().Deployments(ns).List(ctx, listOpts)
	if err != nil {
		return ResourcesResponse{}, fmt.Errorf("list deployments: %w", err)
	}
	for _, d := range deps.Items {
		desired := int32(0)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		out.Deployments = append(out.Deployments, DeploymentInfo{
			Name:              d.Name,
			DesiredReplicas:   desired,
			ReadyReplicas:     d.Status.ReadyReplicas,
			AvailableReplicas: d.Status.AvailableReplicas,
		})
	}

	// Pods
	pods, err := b.KubeClient.CoreV1().Pods(ns).List(ctx, listOpts)
	if err != nil {
		return ResourcesResponse{}, fmt.Errorf("list pods: %w", err)
	}
	for _, p := range pods.Items {
		ready := false
		for _, cond := range p.Status.Conditions {
			if string(cond.Type) == "Ready" && string(cond.Status) == "True" {
				ready = true
			}
		}
		out.Pods = append(out.Pods, PodInfo{
			Name:          p.Name,
			Phase:         string(p.Status.Phase),
			Ready:         ready,
			NodeName:      p.Spec.NodeName,
			RestartCount:  podRestartCount(&p),
			WaitingReason: containerWaitingReason(&p),
		})
	}

	// Services + endpoints
	svcs, err := b.KubeClient.CoreV1().Services(ns).List(ctx, listOpts)
	if err != nil {
		return ResourcesResponse{}, fmt.Errorf("list services: %w", err)
	}
	for _, svc := range svcs.Items {
		ports := make([]int32, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, p.Port)
		}
		hasEP := false
		if ep, epErr := b.KubeClient.CoreV1().Endpoints(ns).Get(ctx, svc.Name, metav1.GetOptions{}); epErr == nil {
			for _, sub := range ep.Subsets {
				if len(sub.Addresses) > 0 {
					hasEP = true
					break
				}
			}
		}
		out.Services = append(out.Services, ServiceInfo{
			Name:         svc.Name,
			ClusterIP:    svc.Spec.ClusterIP,
			HasEndpoints: hasEP,
			Ports:        ports,
		})
	}

	// PVCs — list all in namespace (PVCs may not carry the instance label)
	pvcs, err := b.KubeClient.CoreV1().PersistentVolumeClaims(ns).List(ctx, allOpts)
	if err != nil {
		return ResourcesResponse{}, fmt.Errorf("list pvcs: %w", err)
	}
	for _, pvc := range pvcs.Items {
		cap := ""
		if storage, ok := pvc.Status.Capacity["storage"]; ok {
			cap = storage.String()
		}
		out.PVCs = append(out.PVCs, PVCInfo{
			Name:     pvc.Name,
			Phase:    string(pvc.Status.Phase),
			Capacity: cap,
			Class:    pvcStorageClass(pvc.Spec.StorageClassName),
		})
	}

	// Secrets — key names only, never values
	secrets, err := b.KubeClient.CoreV1().Secrets(ns).List(ctx, allOpts)
	if err != nil {
		return ResourcesResponse{}, fmt.Errorf("list secrets: %w", err)
	}
	for _, s := range secrets.Items {
		keys := make([]string, 0, len(s.Data))
		for k := range s.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.Secrets = append(out.Secrets, SecretMeta{
			Name:     s.Name,
			KeyNames: keys,
		})
	}

	// IngressRoutes (Traefik CRDs) — metadata only
	gvr, err := b.resolveTraefikIngressRouteGVR()
	if err == nil {
		irs, irErr := b.DynamicClient.Resource(gvr).Namespace(ns).List(ctx, allOpts)
		if irErr == nil {
			for _, ir := range irs.Items {
				hosts := extractIngressRouteHosts(ir.Object)
				out.Ingress = append(out.Ingress, IngressInfo{
					Name:  ir.GetName(),
					Kind:  "IngressRoute",
					Hosts: hosts,
				})
			}
		}
	}

	return out, nil
}

func pvcStorageClass(sc *string) string {
	if sc == nil {
		return ""
	}
	return *sc
}

// extractIngressRouteHosts pulls hostname strings from an IngressRoute's routes[].match field.
func extractIngressRouteHosts(obj map[string]any) []string {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return nil
	}
	routes, ok := spec["routes"].([]any)
	if !ok {
		return nil
	}
	var hosts []string
	for _, route := range routes {
		rm, ok := route.(map[string]any)
		if !ok {
			continue
		}
		match, _ := rm["match"].(string)
		if match != "" {
			hosts = append(hosts, match)
		}
	}
	return hosts
}
