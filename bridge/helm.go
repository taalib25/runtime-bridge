package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
)

func (b *Bridge) CreateWorkspace(ctx context.Context, spec WorkspaceSpec) (*release.Release, error) {
	started := time.Now()
	b.Logger.Printf("[CreateWorkspace] Starting for workspace %s, tenant %s", spec.WorkspaceID, spec.TenantID)

	chart, err := loader.Load(b.ChartPath)
	if err != nil {
		b.trackOperation("create", "failure", started)
		b.Logger.Printf("[CreateWorkspace] Failed to load chart: %v", err)
		return nil, fmt.Errorf("load chart: %w", err)
	}

	values, err := b.buildValues(spec)
	if err != nil {
		b.trackOperation("create", "failure", started)
		b.Logger.Printf("[CreateWorkspace] Failed to build values: %v", err)
		return nil, err
	}

	b.Logger.Printf("[CreateWorkspace] Using release name %s, namespace %s", b.releaseName(spec.WorkspaceID), b.workspaceNamespace(spec))

	ns := b.workspaceNamespace(spec)
	helmCfg, err := b.helmConfigForNamespace(ns)
	if err != nil {
		b.trackOperation("create", "failure", started)
		return nil, fmt.Errorf("helm config: %w", err)
	}

	install := action.NewInstall(helmCfg)
	install.ReleaseName = b.releaseName(spec.WorkspaceID)
	install.Namespace = ns
	install.CreateNamespace = spec.CreateNamespace || b.Config.CreateNamespace
	install.SkipCRDs = true
	install.Wait = false

	if authURL := strings.TrimSpace(spec.ForwardAuthURL); authURL != "" {
		if err := b.EnsureForwardAuthMiddleware(ctx, ns, authURL); err != nil {
			b.Logger.Printf("[CreateWorkspace] Warning: failed to create ForwardAuth middleware: %v", err)
		}
	}

	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		b.trackOperation("create", "failure", started)
		b.Logger.Printf("[CreateWorkspace] Helm install failed: %v", err)
		return nil, err
	}

	b.trackOperation("create", "success", started)
	b.Logger.Printf("[CreateWorkspace] Created release %s in %s (took %v)", rel.Name, rel.Namespace, time.Since(started))
	return rel, nil
}

func (b *Bridge) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	started := time.Now()
	helmCfg, err := b.helmConfigForNamespace(workspaceID)
	if err != nil {
		b.trackOperation("delete", "failure", started)
		return fmt.Errorf("helm config: %w", err)
	}
	uninstall := action.NewUninstall(helmCfg)
	uninstall.Wait = false
	_, err = uninstall.Run(b.releaseName(workspaceID))
	if err != nil {
		if strings.Contains(err.Error(), "release: not found") {
			b.trackOperation("delete", "success", started)
			return nil
		}
		b.trackOperation("delete", "failure", started)
		return err
	}
	// Best-effort: remove the ForwardAuth middleware if it exists.
	if mwErr := b.DeleteForwardAuthMiddleware(ctx, workspaceID); mwErr != nil {
		b.Logger.Printf("[DeleteWorkspace] Warning: failed to delete ForwardAuth middleware: %v", mwErr)
	}
	b.Metrics.WorkspaceHealth.DeleteLabelValues(b.ClusterName, workspaceID)
	b.trackOperation("delete", "success", started)
	return nil
}

func (b *Bridge) UpdateWorkspace(ctx context.Context, spec WorkspaceSpec) (*release.Release, error) {
	started := time.Now()
	chart, err := loader.Load(b.ChartPath)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, fmt.Errorf("load chart: %w", err)
	}

	values, err := b.buildValues(spec)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, err
	}

	ns := b.workspaceNamespace(spec)
	helmCfg, err := b.helmConfigForNamespace(ns)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, fmt.Errorf("helm config: %w", err)
	}

	upgrade := action.NewUpgrade(helmCfg)
	upgrade.Namespace = ns
	upgrade.SkipCRDs = true
	upgrade.Wait = false

	if authURL := strings.TrimSpace(spec.ForwardAuthURL); authURL != "" {
		if err := b.EnsureForwardAuthMiddleware(ctx, ns, authURL); err != nil {
			b.Logger.Printf("[UpdateWorkspace] Warning: failed to update ForwardAuth middleware: %v", err)
		}
	}

	rel, err := upgrade.RunWithContext(ctx, b.releaseName(spec.WorkspaceID), chart, values)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, err
	}
	b.trackOperation("update", "success", started)
	return rel, nil
}

func (b *Bridge) ListWorkspaces(ctx context.Context) ([]WorkspaceStatus, error) {
	b.Logger.Printf("[ListWorkspaces] Starting list operation")

	lister := action.NewList(b.HelmConfig)
	lister.All = true
	lister.AllNamespaces = true
	releases, err := lister.Run()
	if err != nil {
		b.Logger.Printf("[ListWorkspaces] Helm list failed: %v", err)
		return nil, err
	}

	b.Logger.Printf("[ListWorkspaces] Found %d total releases", len(releases))

	statuses := make([]WorkspaceStatus, 0, len(releases))
	for _, rel := range releases {
		b.Logger.Printf("[ListWorkspaces] Processing release %s (namespace: %s)", rel.Name, rel.Namespace)
		spec, err := workspaceSpecFromRelease(rel.Name, rel.Config, rel.Namespace, b.ClusterName)
		if err != nil {
			b.Logger.Printf("[ListWorkspaces] Skipping release %s: %v", rel.Name, err)
			continue
		}
		status, err := b.getWorkspaceStatusFromRelease(ctx, rel, spec)
		if err != nil {
			b.Logger.Printf("[ListWorkspaces] Failed to collect workspace status for %s: %v", spec.WorkspaceID, err)
			continue
		}
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].WorkspaceID < statuses[j].WorkspaceID
	})
	b.Logger.Printf("[ListWorkspaces] Returning %d valid workspaces", len(statuses))
	b.Metrics.WorkspaceCount.WithLabelValues(b.ClusterName).Set(float64(len(statuses)))
	return statuses, nil
}

func (b *Bridge) buildValues(spec WorkspaceSpec) (map[string]any, error) {
	spec = b.normalizeWorkspaceSpec(spec)
	repository, tag := splitImageReference(spec.Image)
	if spec.ImageTag != "" {
		tag = spec.ImageTag
	}

	values := map[string]any{
		"fullnameOverride": b.releaseName(spec.WorkspaceID),
		"replicaCount":     1,
		"strategy": map[string]any{
			"type": "Recreate",
		},
		"image": map[string]any{
			"repository": repository,
			"tag":        tag,
		},
		// bootstrap.overwrite controls whether the init container forcefully overwrites
		// HERMES_HOME/config.yaml on every pod start.
		//
		// false (default, pod restarts): preserves agent's runtime config edits and
		//   SOUL.md on the PVC — seeds only if file doesn't exist yet.
		// true (explicit config update via PUT): overwrites config.yaml with the new
		//   Helm-rendered ConfigMap so the backend's config change takes effect.
		//
		// Sessions, memories, logs, skills on the PVC are NEVER touched by bootstrap
		// regardless of this flag — they survive all restarts and image rebuilds.
		"bootstrap": map[string]any{
			"enabled":   true,
			"overwrite": spec.OverwriteConfig,
		},
		// config.values is what the chart's ConfigMap template renders into config.yaml.
		// The backend sends spec.Config as the partial override; chart defaults fill the rest.
		"config": map[string]any{
			"values": spec.Config,
		},
		// env is the chart's flat map of platform env vars (GATEWAY_ALLOW_ALL_USERS, etc.)
		// extraEnv is for arbitrary additional env vars as a list.
		"env":      spec.EnvMap,
		"extraEnv": envValues(spec.Env),
		"secrets":  buildSecrets(spec.Secrets),
		"resources": map[string]any{
			"requests": map[string]any{},
			"limits":   map[string]any{},
		},
		"persistence": map[string]any{
			"enabled": true,
		},
		"service": map[string]any{
			"enabled": true,
			"ports": []any{
				map[string]any{
					"name":       "api-server",
					"port":       8642,
					"targetPort": 8642,
					"protocol":   "TCP",
				},
			},
		},
		"apiServer": map[string]any{
			"enabled":     true,
			"port":        8642,
			"corsOrigins": spec.CORSOrigins,
		},
		"ingress": map[string]any{
			"enabled": spec.ingressEnabled(),
		},
		"bridge":       map[string]any{"workspace": spec},
		"nodeSelector": spec.NodeSelector,
		"tolerations":  spec.Tolerations,
	}

	if policy := strings.TrimSpace(spec.ImagePullPolicy); policy != "" {
		values["image"].(map[string]any)["pullPolicy"] = policy
	}
	if sa := strings.TrimSpace(spec.ServiceAccount); sa != "" {
		values["serviceAccount"] = map[string]any{"name": sa}
	}

	resources := values["resources"].(map[string]any)
	requests := resources["requests"].(map[string]any)
	limits := resources["limits"].(map[string]any)
	if spec.Resources.CPURequest != "" {
		requests["cpu"] = spec.Resources.CPURequest
	}
	if spec.Resources.MemoryRequest != "" {
		requests["memory"] = spec.Resources.MemoryRequest
	}
	if spec.Resources.CPULimit != "" {
		limits["cpu"] = spec.Resources.CPULimit
	}
	if spec.Resources.MemoryLimit != "" {
		limits["memory"] = spec.Resources.MemoryLimit
	}

	persistence := values["persistence"].(map[string]any)
	if spec.Storage.Enabled != nil {
		persistence["enabled"] = *spec.Storage.Enabled
	}
	if spec.Storage.Size != "" {
		persistence["size"] = spec.Storage.Size
	}
	if spec.Storage.StorageClass != "" {
		persistence["storageClass"] = spec.Storage.StorageClass
	}
	if spec.Storage.ExistingClaim != "" {
		persistence["existingClaim"] = spec.Storage.ExistingClaim
	}
	if len(spec.Storage.AccessModes) > 0 {
		persistence["accessModes"] = spec.Storage.AccessModes
	}

	ingress := values["ingress"].(map[string]any)
	className := spec.Network.IngressClassName
	if className == "" {
		className = "traefik"
	}
	ingress["className"] = className
	if host := strings.TrimSpace(spec.Network.host()); host != "" {
		ingress["hosts"] = []map[string]any{{
			"host": host,
			"paths": []map[string]any{{
				"path":        spec.Network.path(),
				"pathType":    "Prefix",
				"servicePort": 8642,
			}},
		}}
	}
	if strings.TrimSpace(spec.ForwardAuthURL) != "" {
		ns := spec.Namespace
		if ns == "" {
			ns = spec.WorkspaceID
		}
		ingress["annotations"] = map[string]any{
			"traefik.ingress.kubernetes.io/router.middlewares": forwardAuthAnnotation(ns),
		}
	}

	return values, nil
}

func workspaceSpecFromRelease(defaultWorkspaceID string, values map[string]any, namespace, clusterName string) (WorkspaceSpec, error) {
	bridgeValues, _ := values["bridge"].(map[string]any)
	workspaceValues, _ := bridgeValues["workspace"].(map[string]any)
	if len(workspaceValues) == 0 {
		return WorkspaceSpec{}, fmt.Errorf("release is missing bridge.workspace metadata")
	}

	spec := WorkspaceSpec{
		WorkspaceID: workspaceString(workspaceValues, "workspaceId", defaultWorkspaceID),
		TenantID:    workspaceString(workspaceValues, "tenantId", ""),
		ClusterID:   workspaceString(workspaceValues, "clusterId", clusterName),
		Namespace:   workspaceString(workspaceValues, "namespace", namespace),
		Image:       workspaceString(workspaceValues, "image", ""),
		ImageTag:    workspaceString(workspaceValues, "imageTag", ""),
		Resources: ResourceSpec{
			CPURequest:    nestedString(workspaceValues, "resources", "cpuRequest"),
			CPULimit:      nestedString(workspaceValues, "resources", "cpuLimit"),
			MemoryRequest: nestedString(workspaceValues, "resources", "memoryRequest"),
			MemoryLimit:   nestedString(workspaceValues, "resources", "memoryLimit"),
		},
		Storage: StorageSpec{
			Size:         nestedString(workspaceValues, "storage", "size"),
			StorageClass: nestedString(workspaceValues, "storage", "storageClass"),
		},
		Network: NetworkSpec{
			Host:             nestedString(workspaceValues, "network", "host"),
			Path:             nestedString(workspaceValues, "network", "path"),
			Subdomain:        nestedString(workspaceValues, "network", "subdomain"),
			IngressClassName: nestedString(workspaceValues, "network", "ingressClassName"),
			Scheme:           nestedString(workspaceValues, "network", "scheme"),
			HealthPath:       nestedString(workspaceValues, "network", "healthPath"),
		},
		HealthCheckPath: workspaceString(workspaceValues, "healthCheckPath", ""),
	}
	if spec.WorkspaceID == "" {
		spec.WorkspaceID = defaultWorkspaceID
	}
	if spec.ClusterID == "" {
		spec.ClusterID = clusterName
	}
	if spec.Namespace == "" {
		spec.Namespace = namespace
	}
	return spec, nil
}

func (b *Bridge) getWorkspaceStatusFromRelease(ctx context.Context, rel *release.Release, spec WorkspaceSpec) (WorkspaceStatus, error) {
	status, err := b.collectWorkspaceStatus(ctx, spec, rel.Name, rel.Info.FirstDeployed.Time)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return status, nil
}

// buildSecrets converts the spec's string secrets map to the any-typed map Helm values expect.
// API_SERVER_KEY must always be present — handler ensures it on create; backend must re-send on update.
func buildSecrets(provided map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range provided {
		out[k] = v
	}
	return out
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func splitImageReference(image string) (string, string) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", ""
	}
	if strings.Contains(image, "@sha256:") {
		return image, ""
	}
	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")
	if lastColon > lastSlash {
		return image[:lastColon], image[lastColon+1:]
	}
	return image, ""
}

func envValues(values []EnvVar) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, item := range values {
		result = append(result, map[string]any{"name": item.Name, "value": item.Value})
	}
	return result
}

func workspaceString(values map[string]any, key, fallback string) string {
	if raw, ok := values[key]; ok {
		if str, ok := raw.(string); ok && strings.TrimSpace(str) != "" {
			return str
		}
	}
	return fallback
}

func nestedString(values map[string]any, parent, key string) string {
	child, _ := values[parent].(map[string]any)
	if child == nil {
		return ""
	}
	return workspaceString(child, key, "")
}

func (s WorkspaceSpec) ingressEnabled() bool {
	if s.IngressEnabled != nil {
		return *s.IngressEnabled
	}
	return strings.TrimSpace(s.Network.Host) != ""
}

func (n NetworkSpec) host() string {
	if strings.TrimSpace(n.Subdomain) != "" && strings.TrimSpace(n.Host) != "" {
		return fmt.Sprintf("%s.%s", n.Subdomain, n.Host)
	}
	return n.Host
}

func (n NetworkSpec) path() string {
	if strings.TrimSpace(n.Path) == "" {
		return "/"
	}
	if strings.HasPrefix(n.Path, "/") {
		return n.Path
	}
	return "/" + n.Path
}

func (b *Bridge) normalizeWorkspaceSpec(spec WorkspaceSpec) WorkspaceSpec {
	spec.WorkspaceID = strings.TrimSpace(spec.WorkspaceID)
	if spec.ClusterID == "" {
		spec.ClusterID = b.ClusterName
	}
	if spec.Namespace == "" {
		spec.Namespace = b.Config.Namespace
	}
	if spec.HealthCheckPath == "" {
		spec.HealthCheckPath = b.Config.HealthPath
	}
	if spec.Network.HealthPath == "" {
		spec.Network.HealthPath = spec.HealthCheckPath
	}
	if spec.Network.Scheme == "" {
		spec.Network.Scheme = "https"
	}
	if spec.Secrets == nil {
		spec.Secrets = map[string]string{}
	}
	if spec.Config == nil {
		spec.Config = map[string]any{}
	}
	// Fall back to the bridge-level default if the caller didn't specify a ForwardAuth URL.
	if strings.TrimSpace(spec.ForwardAuthURL) == "" && strings.TrimSpace(b.Config.DefaultForwardAuthURL) != "" {
		spec.ForwardAuthURL = b.Config.DefaultForwardAuthURL
	}
	return spec
}
