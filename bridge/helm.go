package main

import (
	"context"
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
	chart, err := loader.Load(b.ChartPath)
	if err != nil {
		b.trackOperation("create", "failure", started)
		return nil, fmt.Errorf("load chart: %w", err)
	}

	values, err := b.buildValues(spec)
	if err != nil {
		b.trackOperation("create", "failure", started)
		return nil, err
	}

	install := action.NewInstall(b.HelmConfig)
	install.ReleaseName = b.releaseName(spec.WorkspaceID)
	install.Namespace = b.workspaceNamespace(spec)
	install.CreateNamespace = b.Config.CreateNamespace
	install.Wait = false

	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		b.trackOperation("create", "failure", started)
		return nil, err
	}
	b.trackOperation("create", "success", started)
	return rel, nil
}

func (b *Bridge) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	started := time.Now()
	uninstall := action.NewUninstall(b.HelmConfig)
	uninstall.Wait = false
	_, err := uninstall.Run(b.releaseName(workspaceID))
	if err != nil {
		if strings.Contains(err.Error(), "release: not found") {
			b.trackOperation("delete", "success", started)
			return nil
		}
		b.trackOperation("delete", "failure", started)
		return err
	}
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

	upgrade := action.NewUpgrade(b.HelmConfig)
	upgrade.Namespace = b.workspaceNamespace(spec)
	upgrade.Wait = false

	rel, err := upgrade.RunWithContext(ctx, b.releaseName(spec.WorkspaceID), chart, values)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, err
	}
	b.trackOperation("update", "success", started)
	return rel, nil
}

func (b *Bridge) ListWorkspaces(ctx context.Context) ([]WorkspaceStatus, error) {
	lister := action.NewList(b.HelmConfig)
	lister.All = true
	releases, err := lister.Run()
	if err != nil {
		return nil, err
	}

	statuses := make([]WorkspaceStatus, 0, len(releases))
	for _, rel := range releases {
		spec, err := workspaceSpecFromRelease(rel.Name, rel.Config, rel.Namespace, b.ClusterName)
		if err != nil {
			b.Logger.Printf("Skipping release %s: %v", rel.Name, err)
			continue
		}
		status, err := b.getWorkspaceStatusFromRelease(ctx, rel, spec)
		if err != nil {
			b.Logger.Printf("Failed to collect workspace status for %s: %v", spec.WorkspaceID, err)
			continue
		}
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].WorkspaceID < statuses[j].WorkspaceID
	})
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
		"resources": map[string]any{
			"requests": map[string]any{},
			"limits":   map[string]any{},
		},
		"persistence": map[string]any{
			"enabled": true,
		},
		"ingress": map[string]any{
			"enabled": spec.ingressEnabled(),
		},
		"extraEnv":     envValues(spec.Env),
		"secrets":      spec.Secrets,
		"bridge":       map[string]any{"workspace": spec},
		"config":       spec.Config,
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
	if spec.Network.IngressClassName != "" {
		ingress["className"] = spec.Network.IngressClassName
	}
	if host := strings.TrimSpace(spec.Network.host()); host != "" {
		ingress["hosts"] = []map[string]any{{
			"host": host,
			"paths": []map[string]any{{
				"path":     spec.Network.path(),
				"pathType": "Prefix",
			}},
		}}
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
	return spec
}
