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

const (
	forwardAuthMiddlewareName = "workspace-auth"
	corsMiddlewareName        = "workspace-cors"
)

// traefikMiddlewareGVRs lists Traefik Middleware CRD locations to try in order.
// k3s v1.30+ ships Traefik v2 (traefik.containo.us); some newer builds use v3 (traefik.io).
var traefikMiddlewareGVRs = []schema.GroupVersionResource{
	{Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares"},
	{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "middlewares"},
}

// resolveTraefikGVR returns the first Middleware GVR that exists in the cluster.
func (b *Bridge) resolveTraefikGVR() (schema.GroupVersionResource, error) {
	discovery := b.KubeClient.Discovery()
	for _, gvr := range traefikMiddlewareGVRs {
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
		"Traefik Middleware CRD not found — is Traefik installed and its CRDs registered?",
	)
}

// EnsureForwardAuthMiddleware creates or updates a Traefik Middleware in the workspace
// namespace that proxies every request through authURL for JWT validation.
// authURL must respond to POST with the original request headers and return:
//   - 200 + "Authorization: Bearer <API_SERVER_KEY>" to allow
//   - 401/403 to deny
func (b *Bridge) EnsureForwardAuthMiddleware(ctx context.Context, namespace, authURL string) error {
	gvr, err := b.resolveTraefikGVR()
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gvr.Group + "/" + gvr.Version,
			"kind":       "Middleware",
			"metadata": map[string]any{
				"name":      forwardAuthMiddlewareName,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"forwardAuth": map[string]any{
					// address is your backend's /auth/verify endpoint.
					// Traefik forwards the original request headers (incl. Authorization: Bearer <JWT>).
					"address": authURL,
					// trustForwardHeader passes existing X-Forwarded-* headers to authURL intact.
					"trustForwardHeader": true,
					// authResponseHeaders: Traefik copies these from your auth service's response
					// onto the request before forwarding to the agent. Your /auth/verify endpoint
					// should return "Authorization: Bearer <API_SERVER_KEY>" on success.
					"authResponseHeaders": []any{"Authorization"},
				},
			},
		},
	}

	client := b.DynamicClient.Resource(gvr).Namespace(namespace)
	_, err = client.Create(ctx, obj, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		existing, getErr := client.Get(ctx, forwardAuthMiddlewareName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("get existing middleware: %w", getErr)
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
	}
	return err
}

// DeleteForwardAuthMiddleware removes the ForwardAuth middleware from the workspace namespace.
// Not found is silently ignored.
func (b *Bridge) DeleteForwardAuthMiddleware(ctx context.Context, namespace string) error {
	gvr, err := b.resolveTraefikGVR()
	if err != nil {
		// If Traefik CRDs don't exist, there's nothing to delete.
		b.Logger.Printf("[DeleteForwardAuthMiddleware] Traefik CRD not found, skipping: %v", err)
		return nil
	}
	err = b.DynamicClient.Resource(gvr).Namespace(namespace).Delete(
		ctx, forwardAuthMiddlewareName, metav1.DeleteOptions{},
	)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

// EnsureCORSMiddleware creates or updates a Traefik headers Middleware in the workspace
// namespace that adds CORS response headers for the given origins.
func (b *Bridge) EnsureCORSMiddleware(ctx context.Context, namespace string, origins []string) error {
	gvr, err := b.resolveTraefikGVR()
	if err != nil {
		return err
	}

	// Use customResponseHeaders instead of accessControl* fields so the headers are
	// force-written by Traefik regardless of what the backend pod sends. The accessControl*
	// fields can be overridden or merged when duplicate Middleware CRDs exist across both
	// Traefik API groups (traefik.io and traefik.containo.us), causing stale CORS headers.
	allowOrigin := ""
	if len(origins) > 0 {
		allowOrigin = origins[0]
	}
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gvr.Group + "/" + gvr.Version,
			"kind":       "Middleware",
			"metadata": map[string]any{
				"name":      corsMiddlewareName,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"headers": map[string]any{
					"customResponseHeaders": map[string]any{
						"Access-Control-Allow-Origin":      allowOrigin,
						"Access-Control-Allow-Credentials": "true",
						"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
						"Access-Control-Allow-Headers":     "Authorization,Content-Type,Accept,Accept-Encoding,User-Agent,X-Requested-With,X-Stainless-Lang,X-Stainless-Package-Version,X-Stainless-OS,X-Stainless-Arch,X-Stainless-Runtime,X-Stainless-Runtime-Version,X-Stainless-Retry-Count,X-Stainless-Timeout,OpenAI-Organization,OpenAI-Project",
						"Access-Control-Max-Age":           "86400",
						"Vary":                             "Origin",
					},
				},
			},
		},
	}

	client := b.DynamicClient.Resource(gvr).Namespace(namespace)
	_, err = client.Create(ctx, obj, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		existing, getErr := client.Get(ctx, corsMiddlewareName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("get existing CORS middleware: %w", getErr)
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
	}
	return err
}

// DeleteCORSMiddleware removes the CORS middleware from the workspace namespace.
// Not found is silently ignored.
func (b *Bridge) DeleteCORSMiddleware(ctx context.Context, namespace string) error {
	gvr, err := b.resolveTraefikGVR()
	if err != nil {
		b.Logger.Printf("[DeleteCORSMiddleware] Traefik CRD not found, skipping: %v", err)
		return nil
	}
	err = b.DynamicClient.Resource(gvr).Namespace(namespace).Delete(
		ctx, corsMiddlewareName, metav1.DeleteOptions{},
	)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

// forwardAuthAnnotation returns the Traefik ingress annotation value referencing
// the workspace-auth Middleware in the given namespace.
// Format: "<namespace>-<name>@kubernetescrd"
func forwardAuthAnnotation(namespace string) string {
	return namespace + "-" + forwardAuthMiddlewareName + "@kubernetescrd"
}

// corsAnnotation returns the Traefik ingress annotation value referencing
// the workspace-cors Middleware in the given namespace.
func corsAnnotation(namespace string) string {
	return namespace + "-" + corsMiddlewareName + "@kubernetescrd"
}

// middlewareAnnotations joins middleware annotation values in order.
// CORS must come before ForwardAuth so CORS headers are applied even to ForwardAuth error responses.
func middlewareAnnotations(parts ...string) string {
	return strings.Join(parts, ",")
}
