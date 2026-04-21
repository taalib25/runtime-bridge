package main

import "time"

type WorkspaceSpec struct {
	WorkspaceID      string            `json:"workspaceId"`
	TenantID         string            `json:"tenantId"`
	ClusterID        string            `json:"clusterId,omitempty"`
	Namespace        string            `json:"namespace,omitempty"`
	Image            string            `json:"image"`
	ImageTag         string            `json:"imageTag,omitempty"`
	ImagePullPolicy  string            `json:"imagePullPolicy,omitempty"`
	Resources        ResourceSpec      `json:"resources,omitempty"`
	Storage          StorageSpec       `json:"storage,omitempty"`
	Network          NetworkSpec       `json:"network,omitempty"`
	Env              []EnvVar          `json:"env,omitempty"`
	// EnvMap maps to the chart's flat `env:` block — platform settings like
	// GATEWAY_ALLOW_ALL_USERS, TELEGRAM_ALLOWED_USERS, etc.
	EnvMap           map[string]string `json:"envMap,omitempty"`
	Secrets          map[string]string `json:"secrets,omitempty"`
	// Config maps to chart's config.values — partial Hermes config.yaml override.
	// Chart merges with defaults. Always re-sent on updates; bridge writes it to
	// the ConfigMap which the init container copies to HERMES_HOME/config.yaml.
	Config           map[string]any    `json:"config,omitempty"`
	// OverwriteConfig forces the init container to overwrite HERMES_HOME/config.yaml
	// even if one already exists. Set true only on explicit config updates (PUT).
	// False on normal pod restarts so agent's runtime config edits are preserved.
	OverwriteConfig  bool              `json:"overwriteConfig,omitempty"`
	// CORSOrigins is passed to apiServer.corsOrigins — set to your frontend domain
	// so the browser can call the agent directly without going through the backend.
	// e.g. "https://app.hermeshq.net" or "*" for development.
	CORSOrigins      string            `json:"corsOrigins,omitempty"`
	HealthCheckPath  string            `json:"healthCheckPath,omitempty"`
	ServiceAccount   string            `json:"serviceAccount,omitempty"`
	NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
	Tolerations      []map[string]any  `json:"tolerations,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	IngressEnabled   *bool             `json:"ingressEnabled,omitempty"`
	CreateNamespace  bool              `json:"createNamespace,omitempty"`
	PersistenceClass string            `json:"persistenceClass,omitempty"`
}

type ResourceSpec struct {
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
}

type StorageSpec struct {
	Enabled         *bool    `json:"enabled,omitempty"`
	Size            string   `json:"size,omitempty"`
	StorageClass    string   `json:"storageClass,omitempty"`
	AccessModes     []string `json:"accessModes,omitempty"`
	ExistingClaim   string   `json:"existingClaim,omitempty"`
	MountPath       string   `json:"mountPath,omitempty"`
	VolumeName      string   `json:"volumeName,omitempty"`
	SubPath         string   `json:"subPath,omitempty"`
	RetainOnDelete  bool     `json:"retainOnDelete,omitempty"`
	AdditionalPaths []string `json:"additionalPaths,omitempty"`
}

type NetworkSpec struct {
	Host             string `json:"host,omitempty"`
	Path             string `json:"path,omitempty"`
	Subdomain        string `json:"subdomain,omitempty"`
	IngressClassName string `json:"ingressClassName,omitempty"`
	Scheme           string `json:"scheme,omitempty"`
	HealthPath       string `json:"healthPath,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type WorkspaceStatus struct {
	WorkspaceID      string        `json:"workspaceId"`
	ClusterID        string        `json:"clusterId"`
	ReleaseName      string        `json:"releaseName"`
	Namespace        string        `json:"namespace"`
	Phase            string        `json:"phase"`
	Healthy          bool          `json:"healthy"`
	URL              string        `json:"url,omitempty"`
	Replicas         int32         `json:"replicas"`
	ReadyReplicas    int32         `json:"readyReplicas"`
	PodPhase         string        `json:"podPhase,omitempty"`
	Conditions       []string      `json:"conditions,omitempty"`
	Message          string        `json:"message,omitempty"`
	HealthStatusCode int           `json:"healthStatusCode,omitempty"`
	CreatedAt        time.Time     `json:"createdAt,omitempty"`
	LastCheckedAt    time.Time     `json:"lastCheckedAt,omitempty"`
	Spec             WorkspaceSpec `json:"spec,omitempty"`
}

type Workspace struct {
	Spec   WorkspaceSpec   `json:"spec"`
	Status WorkspaceStatus `json:"status"`
}

type Operation struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	WorkspaceID string     `json:"workspaceId"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Details     string `json:"details,omitempty"`
	OperationID string `json:"operationId,omitempty"`
}
