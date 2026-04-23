package main

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var traefikIngressRouteGVRs = []schema.GroupVersionResource{
	{Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes"},
	{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes"},
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
func (b *Bridge) EnsureIngressRoute(ctx context.Context, namespace, host, serviceName string, servicePort int, hasCORS, hasAuth bool) error {
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

	routes := []any{
		map[string]any{
			"kind":        "Rule",
			"match":       fmt.Sprintf(`Host("%s") && Method("OPTIONS")`, host),
			"priority":    int64(100),
			"middlewares": middlewaresOptions,
			"services":    backend,
		},
		map[string]any{
			"kind":        "Rule",
			"match":       fmt.Sprintf(`Host("%s")`, host),
			"priority":    int64(1),
			"middlewares": middlewaresAll,
			"services":    backend,
		},
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gvr.Group + "/" + gvr.Version,
			"kind":       "IngressRoute",
			"metadata": map[string]any{
				"name":      namespace,
				"namespace": namespace,
			},
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
