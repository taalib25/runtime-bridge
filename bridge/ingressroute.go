package main

import (
	"context"
	"fmt"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var traefikIngressRouteGVRs = []schema.GroupVersionResource{
	{Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes"},
	{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes"},
}

// hostMatchExpr ORs every non-empty, de-duplicated host into a single Traefik match
// expression. Panics-free on an empty slice (returns an empty string — callers always
// pass at least the canonical host).
func hostMatchExpr(hosts []string) string {
	seen := make(map[string]bool, len(hosts))
	var parts []string
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		parts = append(parts, fmt.Sprintf(`Host("%s")`, h))
	}
	return strings.Join(parts, " || ")
}

func (b *Bridge) resolveTraefikIngressRouteGVR() (schema.GroupVersionResource, error) {
	discovery := b.KubeClient.Discovery()
	for _, gvr := range traefikIngressRouteGVRs {
		groups, err := discovery.ServerResourcesForGroupVersion(gvr.Group + "/" + gvr.Version)
		if err != nil {
			continue
		}
		for _, r := range groups.APIResources {
			if r.Name == gvr.Resource {
				return gvr, nil
			}
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf(
		"Traefik IngressRoute CRD not found",
	)
}

// EnsureIngressRoute creates or updates a Traefik IngressRoute for the workspace.
// It creates two routes on the websecure entrypoint:
//  1. OPTIONS requests — CORS middleware only (no ForwardAuth, so browser preflights pass)
//  2. All other requests — CORS + ForwardAuth (JWT validated and swapped)
// ingressRouteMetadata builds the metadata map for an imperative IngressRoute,
// attaching the contract labels/annotations. Empty maps are omitted.
func ingressRouteMetadata(name, namespace string, lbls, anns map[string]string) map[string]any {
	meta := map[string]any{"name": name, "namespace": namespace}
	if len(lbls) > 0 {
		labels := make(map[string]any, len(lbls))
		for k, v := range lbls {
			labels[k] = v
		}
		meta["labels"] = labels
	}
	if len(anns) > 0 {
		annotations := make(map[string]any, len(anns))
		for k, v := range anns {
			annotations[k] = v
		}
		meta["annotations"] = annotations
	}
	return meta
}

// EnsureIngressRoute creates or updates a Traefik IngressRoute for the workspace.
// hosts[0] is the canonical host; any additional entries (e.g. the cluster-specific
// host the public redirector sends browsers to) are OR'd into the same match rule so
// one IngressRoute serves every hostname that should reach this instance.
func (b *Bridge) EnsureIngressRoute(ctx context.Context, namespace string, hosts []string, serviceName string, servicePort int, hasCORS, hasAuth bool, lbls, anns map[string]string) error {
	gvr, err := b.resolveTraefikIngressRouteGVR()
	if err != nil {
		return err
	}

	var middlewaresAll, middlewaresOptions []any

	if hasCORS {
		middlewaresAll = append(middlewaresAll, map[string]any{
			"name":      corsMiddlewareName,
			"namespace": namespace,
		})
		middlewaresOptions = append(middlewaresOptions, map[string]any{
			"name":      corsMiddlewareName,
			"namespace": namespace,
		})
	}
	if hasAuth {
		middlewaresAll = append(middlewaresAll, map[string]any{
			"name":      forwardAuthMiddlewareName,
			"namespace": namespace,
		})
		// OPTIONS intentionally excludes ForwardAuth so preflight succeeds
	}

	backend := []any{map[string]any{
		"name": serviceName,
		"port": int64(servicePort),
	}}

	hostMatch := hostMatchExpr(hosts)
	routes := []any{
		map[string]any{
			"kind":        "Rule",
			"match":       fmt.Sprintf(`(%s) && Method("OPTIONS")`, hostMatch),
			"priority":    int64(100),
			"middlewares": middlewaresOptions,
			"services":    backend,
		},
		map[string]any{
			"kind":        "Rule",
			"match":       hostMatch,
			"priority":    int64(1),
			"middlewares": middlewaresAll,
			"services":    backend,
		},
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gvr.Group + "/" + gvr.Version,
			"kind":       "IngressRoute",
			"metadata": ingressRouteMetadata(namespace, namespace, lbls, anns),
			"spec": map[string]any{
				"entryPoints": []any{"websecure"},
				"routes":      routes,
				"tls": map[string]any{
					"certResolver": "letsencrypt",
				},
			},
		},
	}

	client := b.DynamicClient.Resource(gvr).Namespace(namespace)
	_, err = client.Create(ctx, obj, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		existing, getErr := client.Get(ctx, namespace, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("get existing IngressRoute: %w", getErr)
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
	}
	return err
}

// DeleteIngressRoute removes the workspace IngressRoute. Not found is silently ignored.
func (b *Bridge) DeleteIngressRoute(ctx context.Context, namespace string) error {
	gvr, err := b.resolveTraefikIngressRouteGVR()
	if err != nil {
		b.Logger.Printf("[DeleteIngressRoute] Traefik CRD not found, skipping: %v", err)
		return nil
	}
	err = b.DynamicClient.Resource(gvr).Namespace(namespace).Delete(
		ctx, namespace, metav1.DeleteOptions{},
	)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}
