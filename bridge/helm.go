package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	createNS := spec.CreateNamespace || b.Config.CreateNamespace
	if createNS {
		if err := b.ensureNamespace(ctx, ns); err != nil {
			b.trackOperation("create", "failure", started)
			return nil, fmt.Errorf("ensure namespace: %w", err)
		}
	}

	install := action.NewInstall(helmCfg)
	install.ReleaseName = b.releaseName(spec.WorkspaceID)
	install.Namespace = ns
	install.CreateNamespace = createNS
	install.SkipCRDs = true
	install.Wait = false

	hasCORS := len(parseCORSOrigins(spec.CORSOrigins)) > 0
	hasAuth := strings.TrimSpace(spec.ForwardAuthURL) != ""

	if hasAuth {
		if err := b.EnsureForwardAuthMiddleware(ctx, ns, strings.TrimSpace(spec.ForwardAuthURL)); err != nil {
			b.Logger.Printf("[CreateWorkspace] Warning: failed to create ForwardAuth middleware: %v", err)
		}
	}
	if hasCORS {
		if err := b.EnsureCORSMiddleware(ctx, ns, parseCORSOrigins(spec.CORSOrigins)); err != nil {
			b.Logger.Printf("[CreateWorkspace] Warning: failed to create CORS middleware: %v", err)
		}
	}

	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		b.trackOperation("create", "failure", started)
		b.Logger.Printf("[CreateWorkspace] Helm install failed: %v", err)
		return nil, err
	}

	if hasCORS || hasAuth {
		host := spec.Network.host()
		if err := b.EnsureIngressRoute(ctx, ns, host, ns, 8642, hasCORS, hasAuth); err != nil {
			b.Logger.Printf("[CreateWorkspace] Warning: failed to create IngressRoute: %v", err)
		} else {
			// Remove the Helm-managed Ingress so only IngressRoute routes this host.
			b.deleteHelmIngress(ctx, ns)
		}
	}

	if spec.DashboardEnabled {
		if !hasAuth {
			b.Logger.Printf("[CreateWorkspace] WARNING: dashboardEnabled=true but ForwardAuthURL not set — dash-%s is publicly accessible", spec.Network.host())
		}
		if dHost := dashboardHost(spec); dHost != "" {
			if err := b.EnsureDashboardIngressRoute(ctx, ns, dHost, ns, hasCORS, hasAuth); err != nil {
				b.Logger.Printf("[CreateWorkspace] Warning: failed to create dashboard IngressRoute: %v", err)
			}
		}
	}

	b.trackOperation("create", "success", started)
	b.Logger.Printf("[CreateWorkspace] Created release %s in %s (took %v)", rel.Name, rel.Namespace, time.Since(started))
	return rel, nil
}

// deleteHelmIngress removes the Kubernetes Ingress created by the Helm chart so the
// IngressRoute takes exclusive control of routing for this workspace.
func (b *Bridge) deleteHelmIngress(ctx context.Context, namespace string) {
	_, err := b.KubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return
	}
	if err := b.KubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		b.Logger.Printf("[deleteHelmIngress] Warning: failed to delete Ingress %s: %v", namespace, err)
	}
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
	// Best-effort: remove Traefik resources if they exist.
	if mwErr := b.DeleteIngressRoute(ctx, workspaceID); mwErr != nil {
		b.Logger.Printf("[DeleteWorkspace] Warning: failed to delete IngressRoute: %v", mwErr)
	}
	if mwErr := b.DeleteDashboardIngressRoute(ctx, workspaceID); mwErr != nil {
		b.Logger.Printf("[DeleteWorkspace] Warning: failed to delete dashboard IngressRoute: %v", mwErr)
	}
	if mwErr := b.DeleteForwardAuthMiddleware(ctx, workspaceID); mwErr != nil {
		b.Logger.Printf("[DeleteWorkspace] Warning: failed to delete ForwardAuth middleware: %v", mwErr)
	}
	if mwErr := b.DeleteCORSMiddleware(ctx, workspaceID); mwErr != nil {
		b.Logger.Printf("[DeleteWorkspace] Warning: failed to delete CORS middleware: %v", mwErr)
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

	hasCORS := len(parseCORSOrigins(spec.CORSOrigins)) > 0
	hasAuth := strings.TrimSpace(spec.ForwardAuthURL) != ""

	if hasAuth {
		if err := b.EnsureForwardAuthMiddleware(ctx, ns, strings.TrimSpace(spec.ForwardAuthURL)); err != nil {
			b.Logger.Printf("[UpdateWorkspace] Warning: failed to update ForwardAuth middleware: %v", err)
		}
	}
	if hasCORS {
		if err := b.EnsureCORSMiddleware(ctx, ns, parseCORSOrigins(spec.CORSOrigins)); err != nil {
			b.Logger.Printf("[UpdateWorkspace] Warning: failed to update CORS middleware: %v", err)
		}
	}

	rel, err := upgrade.RunWithContext(ctx, b.releaseName(spec.WorkspaceID), chart, values)
	if err != nil {
		b.trackOperation("update", "failure", started)
		return nil, err
	}

	if hasCORS || hasAuth {
		host := spec.Network.host()
		if err := b.EnsureIngressRoute(ctx, ns, host, ns, 8642, hasCORS, hasAuth); err != nil {
			b.Logger.Printf("[UpdateWorkspace] Warning: failed to update IngressRoute: %v", err)
		} else {
			b.deleteHelmIngress(ctx, ns)
		}
	}

	if spec.DashboardEnabled {
		if !hasAuth {
			b.Logger.Printf("[UpdateWorkspace] WARNING: dashboardEnabled=true but ForwardAuthURL not set — dash-%s is publicly accessible", spec.Network.host())
		}
		if dHost := dashboardHost(spec); dHost != "" {
			if err := b.EnsureDashboardIngressRoute(ctx, ns, dHost, ns, hasCORS, hasAuth); err != nil {
				b.Logger.Printf("[UpdateWorkspace] Warning: failed to update dashboard IngressRoute: %v", err)
			}
		}
	} else {
		_ = b.DeleteDashboardIngressRoute(ctx, ns)
	}

	b.trackOperation("update", "success", started)
	return rel, nil
}

func (b *Bridge) ListWorkspaces(ctx context.Context) ([]WorkspaceStatus, error) {
	b.Logger.Printf("[ListWorkspaces] Starting list operation")

	// Use an empty-namespace config so the secret driver queries across all
	// namespaces. The bridge's default HelmConfig is scoped to hermes-bridge
	// and cannot see releases installed into workspace namespaces.
	allNsCfg, err := newHelmActionConfigForNamespace(b.Config, "")
	if err != nil {
		return nil, fmt.Errorf("helm config for all-namespace list: %w", err)
	}
	lister := action.NewList(allNsCfg)
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
	repository, splitTag := splitImageReference(spec.Image)
	// ImageTag from spec takes precedence; fall back to the tag embedded in the image
	// reference, then to the normalizeWorkspaceSpec default ("latest").
	tag := spec.ImageTag
	if tag == "" {
		tag = splitTag
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
		// The backend sends spec.HermesConfig as the partial override; chart defaults fill the rest.
		"config": map[string]any{
			"values": hermesConfigToMap(spec.HermesConfig),
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
			"ports": func() []any {
				ports := []any{map[string]any{
					"name": "api-server", "port": 8642, "targetPort": 8642, "protocol": "TCP",
				}}
				if spec.DashboardEnabled {
					ports = append(ports, map[string]any{
						"name": "dashboard", "port": int64(9119), "targetPort": int64(9119), "protocol": "TCP",
					})
				}
				return ports
			}(),
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

	if spec.Plan != "" {
		values["podLabels"] = map[string]any{"hermes.ai/plan": spec.Plan}
	}

	if spec.DashboardEnabled {
		// GATEWAY_HEALTH_URL: pod shares network namespace, so 127.0.0.1 reaches the gateway container
		dashEnv := []any{
			map[string]any{"name": "GATEWAY_HEALTH_URL", "value": "http://127.0.0.1:8642"},
		}
		for k, v := range spec.EnvMap {
			dashEnv = append(dashEnv, map[string]any{"name": k, "value": v})
		}
		values["extraContainers"] = []any{map[string]any{
			"name":  "dashboard",
			"image": repository + ":" + tag,
			// args (not command) — entrypoint is already `hermes`; args selects the subcommand
			"args": []any{"dashboard", "--host", "0.0.0.0", "--port", "9119", "--no-open", "--insecure"},
			"ports": []any{map[string]any{
				"name": "dashboard", "containerPort": int64(9119), "protocol": "TCP",
			}},
			"env": dashEnv,
			"volumeMounts": []any{map[string]any{
				"name": "data", "mountPath": "/opt/data",
			}},
		}}
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
	{
		ns := spec.Namespace
		if ns == "" {
			ns = spec.WorkspaceID
		}
		var middlewareParts []string
		if strings.TrimSpace(spec.CORSOrigins) != "" {
			middlewareParts = append(middlewareParts, corsAnnotation(ns))
		}
		if strings.TrimSpace(spec.ForwardAuthURL) != "" {
			middlewareParts = append(middlewareParts, forwardAuthAnnotation(ns))
		}
		if len(middlewareParts) > 0 {
			ingress["annotations"] = map[string]any{
				"traefik.ingress.kubernetes.io/router.middlewares": middlewareAnnotations(middlewareParts...),
			}
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
	// Default image to the official Hermes agent.
	if strings.TrimSpace(spec.Image) == "" {
		spec.Image = "nousresearch/hermes-agent"
	}
	// Default tag to "latest" only when the caller didn't embed a tag in the image
	// reference (e.g. "image:v1.2") and didn't set ImageTag explicitly.
	if strings.TrimSpace(spec.ImageTag) == "" {
		_, embeddedTag := splitImageReference(spec.Image)
		if embeddedTag == "" {
			spec.ImageTag = "latest"
		}
	}
	// Default namespace to the workspace ID (one namespace per tenant).
	if spec.Namespace == "" {
		if spec.WorkspaceID != "" {
			spec.Namespace = spec.WorkspaceID
		} else {
			spec.Namespace = b.Config.Namespace
		}
	}
	// Auto-create namespace when it's isolated per workspace.
	if !spec.CreateNamespace && spec.Namespace == spec.WorkspaceID {
		spec.CreateNamespace = true
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
	// Derive host from workspaceID + default domain when not explicitly set.
	if spec.Network.Host == "" && strings.TrimSpace(b.Config.DefaultDomain) != "" {
		spec.Network.Host = spec.WorkspaceID + "." + b.Config.DefaultDomain
	}
	// Default ingress to enabled whenever a host is configured.
	if spec.IngressEnabled == nil && spec.Network.Host != "" {
		t := true
		spec.IngressEnabled = &t
	}
	if spec.Secrets == nil {
		spec.Secrets = map[string]string{}
	}
	// Inject workspace identity into pod env so the agent knows its context.
	if spec.EnvMap == nil {
		spec.EnvMap = map[string]string{}
	}
	spec.EnvMap["WORKSPACE_ID"] = spec.WorkspaceID
	spec.EnvMap["TENANT_ID"] = spec.TenantID
	if spec.Plan != "" {
		spec.EnvMap["PLAN"] = spec.Plan
	}
	// Fall back to the bridge-level default if the caller didn't specify a ForwardAuth URL.
	if strings.TrimSpace(spec.ForwardAuthURL) == "" && strings.TrimSpace(b.Config.DefaultForwardAuthURL) != "" {
		spec.ForwardAuthURL = b.Config.DefaultForwardAuthURL
	}
	// Fall back to the bridge-level default if the caller didn't specify CORS origins.
	if strings.TrimSpace(spec.CORSOrigins) == "" && strings.TrimSpace(b.Config.DefaultCORSOrigins) != "" {
		spec.CORSOrigins = b.Config.DefaultCORSOrigins
	}
	return spec
}

// hermesConfigToMap converts the typed HermesConfig into the map[string]any
// that Helm chart values expect under config.values. Fields with zero/nil values
// are omitted so chart defaults take effect.
func hermesConfigToMap(cfg HermesConfig) map[string]any {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// isEmptyHermesConfig reports whether the caller sent no config fields at all,
// meaning we should not overwrite the agent's runtime config on the PVC.
func isEmptyHermesConfig(cfg HermesConfig) bool {
	return cfg.Model == nil &&
		cfg.Agent == nil &&
		cfg.Terminal == nil &&
		cfg.Display == nil &&
		cfg.Browser == nil &&
		cfg.Memory == nil &&
		cfg.Compression == nil &&
		cfg.Security == nil &&
		cfg.Voice == nil &&
		cfg.Auxiliary == nil &&
		cfg.Gateway == nil &&
		cfg.Soul == nil
}

// parseCORSOrigins splits a comma-separated origins string into a trimmed slice.
// Returns nil if s is blank.
func parseCORSOrigins(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
