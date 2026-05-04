package main

import "time"

type WorkspaceSpec struct {
	WorkspaceID     string            `json:"workspaceId"`
	TenantID        string            `json:"tenantId"`
	ClusterID       string            `json:"clusterId,omitempty"`
	Namespace       string            `json:"namespace,omitempty"`
	Image           string            `json:"image,omitempty"`
	ImageTag        string            `json:"imageTag,omitempty"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Resources       ResourceSpec      `json:"resources,omitempty"`
	Storage         StorageSpec       `json:"storage,omitempty"`
	Network         NetworkSpec       `json:"network,omitempty"`
	Env             []EnvVar          `json:"env,omitempty"`
	// EnvMap maps to the chart's flat env: block — platform switches like
	// GATEWAY_ALLOW_ALL_USERS, WHATSAPP_ENABLED, TELEGRAM_ALLOWED_USERS, etc.
	EnvMap map[string]string `json:"envMap,omitempty"`
	// Secrets are injected as a Kubernetes Secret. API_SERVER_KEY is auto-generated
	// on create; re-send it unchanged on every PUT or it will be rotated.
	Secrets map[string]string `json:"secrets,omitempty"`
	// HermesConfig is the partial Hermes config.yaml override for this workspace.
	// Only set sections are applied; chart defaults fill the rest.
	HermesConfig HermesConfig `json:"config,omitempty"`
	// OverwriteConfig forces the init container to overwrite HERMES_HOME/config.yaml
	// on the PVC. Set true only on explicit config updates (PUT). False on restarts
	// preserves agent's runtime edits. Bridge sets this automatically based on whether
	// the config block is present in the request.
	OverwriteConfig bool `json:"overwriteConfig,omitempty"`
	// CORSOrigins is the comma-separated list of browser origins allowed to call the
	// agent's API server directly. Bridge-level default applies when omitted.
	// e.g. "https://app.hermeshq.net,http://localhost:3002"
	CORSOrigins string `json:"corsOrigins,omitempty"`
	// ForwardAuthURL enables Traefik ForwardAuth on this workspace's ingress.
	// Traefik calls this URL for every request; your endpoint validates the JWT and
	// returns 200 + "Authorization: Bearer <API_SERVER_KEY>" to allow, 401/403 to deny.
	// Bridge-level default applies when omitted.
	ForwardAuthURL  string            `json:"forwardAuthURL,omitempty"`
	HealthCheckPath string            `json:"healthCheckPath,omitempty"`
	ServiceAccount  string            `json:"serviceAccount,omitempty"`
	NodeSelector    map[string]string `json:"nodeSelector,omitempty"`
	Tolerations     []map[string]any  `json:"tolerations,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	// IngressEnabled defaults to true. Set explicitly to false to disable.
	IngressEnabled   *bool  `json:"ingressEnabled,omitempty"`
	CreateNamespace  bool   `json:"createNamespace,omitempty"`
	PersistenceClass string `json:"persistenceClass,omitempty"`
	// Plan is the pricing tier: "free", "pro", "enterprise".
	// Surfaces as pod label hermes.ai/plan for metrics and cost attribution.
	Plan string `json:"plan,omitempty"`
	// DashboardEnabled starts a hermes dashboard sidecar (port 9119) and creates
	// a second IngressRoute at dash-{ws-id}.{domain}.
	// WARNING: the dashboard has no built-in auth — set ForwardAuthURL to protect it.
	DashboardEnabled bool `json:"dashboardEnabled,omitempty"`
	// RuntimeMode selects the container runtime: "hermes-agent" (default) or "webui".
	// "webui" deploys ghcr.io/nesquena/hermes-webui — a single-container mode where
	// the WebUI runs the Hermes agent in-process and serves a browser-based interface.
	RuntimeMode string `json:"runtimeMode,omitempty"`
	// RuntimePort is the container port the runtime listens on.
	// Defaults to 8642 for hermes-agent, 8787 for webui.
	RuntimePort int `json:"runtimePort,omitempty"`
}

// ─── Hermes config.yaml ──────────────────────────────────────────────────────

// HermesConfig is the partial Hermes config.yaml sent per-workspace.
// All sections are optional; omit any section to use chart defaults.
// Field names match config.yaml top-level keys exactly.
//
// Backend DB recommendation: store the full HermesConfig JSON alongside
// workspaceId, tenantId, API_SERVER_KEY, and the chosen model.Default so you
// can re-send the complete spec on updates without reading from the cluster.
type HermesConfig struct {
	Model       *ModelConfig       `json:"model,omitempty"`
	Agent       *AgentConfig       `json:"agent,omitempty"`
	Terminal    *TerminalConfig    `json:"terminal,omitempty"`
	Display     *DisplayConfig     `json:"display,omitempty"`
	Browser     *BrowserConfig     `json:"browser,omitempty"`
	Memory      *MemoryConfig      `json:"memory,omitempty"`
	Compression *CompressionConfig `json:"compression,omitempty"`
	Security    *SecurityConfig    `json:"security,omitempty"`
	Voice       *VoiceConfig       `json:"voice,omitempty"`
	Auxiliary   *AuxiliaryConfig   `json:"auxiliary,omitempty"`
	Gateway     *GatewayConfig     `json:"gateway,omitempty"`
	Soul        *SoulConfig        `json:"soul,omitempty"`
}

// ModelConfig selects the LLM the agent uses.
// Store model.Default in your DB — it's the key differentiator between workspace tiers.
type ModelConfig struct {
	// Default is the model ID. e.g. "anthropic/claude-opus-4-7", "openai/gpt-4o",
	// "google/gemini-2.5-pro", "deepseek/deepseek-r1", "nous/hermes-3-405b"
	Default string `json:"default,omitempty"`
	// Provider: "auto" (default), "anthropic", "openrouter", "openai", "nous", "google", etc.
	Provider string `json:"provider,omitempty"`
	// BaseURL overrides the provider API endpoint. Leave empty for the provider default.
	BaseURL string `json:"base_url,omitempty"`
	// ContextLength caps the model's context window (tokens). 0 = use model default.
	ContextLength int `json:"context_length,omitempty"`
	// MaxTokens caps the per-response token budget. 0 = use model default.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// AgentConfig controls agent behaviour and limits.
type AgentConfig struct {
	// MaxTurns is the max agent turns per session before it stops (default 90).
	MaxTurns int `json:"max_turns,omitempty"`
	// GatewayTimeout is the request timeout for the gateway in seconds (default 1800).
	GatewayTimeout int `json:"gateway_timeout,omitempty"`
	// RestartDrainTimeout is seconds to drain in-flight requests before a restart (default 60).
	RestartDrainTimeout int `json:"restart_drain_timeout,omitempty"`
	// ToolUseEnforcement: "auto" (default), "always", "never".
	ToolUseEnforcement string `json:"tool_use_enforcement,omitempty"`
	// ReasoningEffort: "none", "low", "minimal", "medium" (default), "high", "xhigh".
	// Only used by reasoning-capable models (o1, R1, etc.).
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// TerminalConfig controls the agent's shell/code-execution backend.
type TerminalConfig struct {
	// Backend: "local" (default), "ssh", "docker", "singularity", "modal", "daytona".
	Backend string `json:"backend,omitempty"`
	// Cwd is the working directory inside the container (default ".").
	Cwd string `json:"cwd,omitempty"`
	// Timeout is the per-command shell timeout in seconds (default 180).
	Timeout int `json:"timeout,omitempty"`
	// PersistentShell keeps the shell process alive between commands (default true).
	PersistentShell *bool `json:"persistent_shell,omitempty"`
	// EnvPassthrough lists env var names to forward into the terminal session.
	EnvPassthrough []string `json:"env_passthrough,omitempty"`

	// Docker-specific (backend: "docker")
	DockerImage          string   `json:"docker_image,omitempty"`
	DockerVolumes        []string `json:"docker_volumes,omitempty"`
	DockerForwardEnv     []string `json:"docker_forward_env,omitempty"`
	ContainerCPU         int      `json:"container_cpu,omitempty"`
	ContainerMemoryMB    int      `json:"container_memory,omitempty"`
	ContainerDiskMB      int      `json:"container_disk,omitempty"`
	ContainerPersistent  *bool    `json:"container_persistent,omitempty"`

	// SSH-specific (backend: "ssh")
	SSHHost     string `json:"ssh_host,omitempty"`
	SSHPort     int    `json:"ssh_port,omitempty"`
	SSHUser     string `json:"ssh_user,omitempty"`
	SSHKeyPath  string `json:"ssh_key_path,omitempty"`
}

// DisplayConfig controls what the agent streams to the user.
type DisplayConfig struct {
	// ToolProgress: "all" (default), "minimal", "none".
	ToolProgress string `json:"tool_progress,omitempty"`
	// InterimAssistantMessages streams partial assistant messages while reasoning.
	InterimAssistantMessages *bool `json:"interim_assistant_messages,omitempty"`
}

// BrowserConfig controls the agent's headless browser integration.
type BrowserConfig struct {
	// InactivityTimeout is seconds of browser idle time before auto-close (default 120).
	InactivityTimeout int `json:"inactivity_timeout,omitempty"`
	// CommandTimeout is the per-browser-command timeout in seconds (default 30).
	CommandTimeout int `json:"command_timeout,omitempty"`
	// AllowPrivateURLs lets the agent browse localhost / RFC-1918 addresses.
	AllowPrivateURLs *bool `json:"allow_private_urls,omitempty"`
	// RecordSessions saves WebM recordings to HERMES_HOME/browser_recordings/.
	RecordSessions *bool `json:"record_sessions,omitempty"`
}

// MemoryConfig controls the agent's persistent user/session memory.
type MemoryConfig struct {
	// Enabled toggles memory persistence (default true).
	Enabled *bool `json:"memory_enabled,omitempty"`
	// UserProfileEnabled builds a persisted user profile (default true).
	UserProfileEnabled *bool `json:"user_profile_enabled,omitempty"`
	// MemoryCharLimit caps the memory store in characters (default 2200).
	MemoryCharLimit int `json:"memory_char_limit,omitempty"`
	// UserCharLimit caps the user profile in characters (default 1375).
	UserCharLimit int `json:"user_char_limit,omitempty"`
}

// CompressionConfig controls automatic context-window compression.
type CompressionConfig struct {
	// Enabled toggles compression (default true).
	Enabled *bool `json:"enabled,omitempty"`
	// Threshold triggers compression when context reaches this fraction of the window (default 0.5).
	Threshold float64 `json:"threshold,omitempty"`
	// TargetRatio is the post-compression target fraction (default 0.2).
	TargetRatio float64 `json:"target_ratio,omitempty"`
	// ProtectLastN keeps the last N messages from being compressed (default 20).
	ProtectLastN int `json:"protect_last_n,omitempty"`
}

// SecurityConfig controls secret redaction and command policy enforcement.
type SecurityConfig struct {
	// RedactSecrets auto-redacts API keys and passwords in agent output (default true).
	RedactSecrets *bool `json:"redact_secrets,omitempty"`
	// TirithEnabled enables the Tirith command-safety scanner (default true).
	TirithEnabled *bool `json:"tirith_enabled,omitempty"`
	// TirithFailOpen allows commands through when Tirith is unavailable (default true).
	TirithFailOpen *bool `json:"tirith_fail_open,omitempty"`
	// TirithTimeout is the seconds to wait for a Tirith scan result (default 5).
	TirithTimeout int `json:"tirith_timeout,omitempty"`
}

// VoiceConfig controls text-to-speech and speech-to-text.
// Keys in Secrets must supply the relevant provider API key.
type VoiceConfig struct {
	// TTSProvider: "edge" (free, default), "elevenlabs", "openai", "neutts".
	TTSProvider string `json:"tts_provider,omitempty"`
	TTSModel    string `json:"tts_model,omitempty"`
	TTSVoice    string `json:"tts_voice,omitempty"`
	// STTProvider: "local" (faster-whisper, default), "groq", "openai".
	STTProvider string `json:"stt_provider,omitempty"`
	STTModel    string `json:"stt_model,omitempty"`
}

// AuxiliaryConfig selects the lightweight/specialist models used for non-chat tasks.
// Leave any section nil to use the primary model for that task.
type AuxiliaryConfig struct {
	Vision      *AuxModelConfig `json:"vision,omitempty"`
	WebExtract  *AuxModelConfig `json:"web_extract,omitempty"`
	Compression *AuxModelConfig `json:"compression,omitempty"`
	Approval    *AuxModelConfig `json:"approval,omitempty"`
}

// AuxModelConfig selects a model for one auxiliary task.
type AuxModelConfig struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
	// Timeout is the request timeout in seconds for this auxiliary task.
	Timeout int `json:"timeout,omitempty"`
}

// GatewayConfig controls the API gateway session and multi-user behaviour.
type GatewayConfig struct {
	// GroupSessionsPerUser creates one session per user in group channels (default true).
	GroupSessionsPerUser *bool `json:"group_sessions_per_user,omitempty"`
	// SessionResetPolicy: "idle" (default) resets after inactivity; "daily" resets at midnight.
	SessionResetPolicy string `json:"session_reset_policy,omitempty"`
	// SessionResetTimeout is the idle minutes before a session is reset (default 1440 = 24h).
	SessionResetTimeout int `json:"session_reset_timeout,omitempty"`
}

// SoulConfig is the agent's SOUL.md — its system prompt and personality.
// Set text to override the chart default for this workspace.
// Store in your DB if you want per-tenant personalities.
type SoulConfig struct {
	Text string `json:"text,omitempty"`
}

// ─── Infrastructure types ─────────────────────────────────────────────────────

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

// ─── API response types ───────────────────────────────────────────────────────

type WorkspaceStatus struct {
	WorkspaceID      string        `json:"workspaceId"`
	ClusterID        string        `json:"clusterId"`
	ReleaseName      string        `json:"releaseName"`
	Namespace        string        `json:"namespace"`
	Phase            string        `json:"phase"`
	Healthy          bool          `json:"healthy"`
	URL              string        `json:"url,omitempty"`
	DashboardURL     string        `json:"dashboardURL,omitempty"`
	Replicas         int32         `json:"replicas"`
	ReadyReplicas    int32         `json:"readyReplicas"`
	PodPhase         string        `json:"podPhase,omitempty"`
	Conditions       []string      `json:"conditions,omitempty"`
	Message          string        `json:"message,omitempty"`
	HealthStatusCode int           `json:"healthStatusCode,omitempty"`
	// WaitingReason is the container Waiting.Reason from Kubernetes when a
	// container is not running. Common values: CrashLoopBackOff, ImagePullBackOff,
	// ErrImagePull, OOMKilled, CreateContainerConfigError. Empty when healthy.
	WaitingReason string `json:"waitingReason,omitempty"`
	// RestartCount is the total restart count across all containers in the pod.
	// A non-zero value indicates the container has crashed at least once.
	RestartCount  int32     `json:"restartCount,omitempty"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	LastCheckedAt time.Time `json:"lastCheckedAt,omitempty"`
	Spec          WorkspaceSpec `json:"spec,omitempty"`
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
