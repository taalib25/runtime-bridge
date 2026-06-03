package main

import (
	"encoding/json"
	"time"
)

type InstanceSpec struct {
	InstanceID string `json:"instanceId"`
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
	// CommonLabels and CommonAnnotations are backend-owned business metadata under
	// the hermescloud.dev/* prefix, applied to every resource the bridge creates.
	// The bridge validates and persists them but never invents or mutates them.
	// See docs/label-contract.md.
	CommonLabels      map[string]string `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`
	// IngressEnabled defaults to true. Set explicitly to false to disable.
	IngressEnabled   *bool  `json:"ingressEnabled,omitempty"`
	CreateNamespace  bool   `json:"createNamespace,omitempty"`
	PersistenceClass string `json:"persistenceClass,omitempty"`
	// Plan is the pricing tier determined by the backend (e.g. "free", "pro").
	// For resource labelling, backend sends it as hermescloud.dev/plan in CommonLabels
	// (see docs/label-contract.md); this field is kept for the PLAN env var / metrics.
	// Not validated by the bridge.
	Plan string `json:"plan,omitempty"`
	// RuntimeMode must be "runtime-node-core" or empty (treated as "runtime-node-core").
	// Any other value is rejected with a 400. The bridge only supports the prebuilt
	// runtime-node-core image (ghcr.io/taalib25/runtime-node-core) which bundles
	// hermes-webui + hermes-agent and runs as hermeswebui (UID 1024).
	RuntimeMode string `json:"runtimeMode,omitempty"`
	// RuntimePort is the container port the runtime listens on.
	// Defaults to 8787 (runtime-node-core listens on 8787).
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
	Session     *SessionConfig     `json:"session,omitempty"`
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

// GatewayConfig controls multi-user session isolation.
// Maps to the gateway: section in config.yaml.
type GatewayConfig struct {
	// GroupSessionsPerUser creates one conversation session per user in group channels (default true).
	GroupSessionsPerUser *bool `json:"group_sessions_per_user,omitempty"`
}

// SessionConfig controls when the agent resets its conversation context.
// Maps to the session: section in config.yaml (separate from gateway:).
type SessionConfig struct {
	// ResetPolicy: "idle_timeout" | "daily" | "idle_and_daily" | "manual" (default "idle_and_daily").
	ResetPolicy string `json:"reset_policy,omitempty"`
	// IdleTimeoutHours resets the session after this many hours of inactivity (default 8).
	IdleTimeoutHours int `json:"idle_timeout_hours,omitempty"`
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

type InstanceStatus struct {
	InstanceID string `json:"instanceId"`
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
	Spec InstanceSpec `json:"spec,omitempty"`
	// SelectorLabels, CommonLabels, and CommonAnnotations echo the effective label
	// contract applied to this instance, so the backend can verify round-trip and
	// detect drift. See docs/label-contract.md.
	SelectorLabels    map[string]string `json:"selectorLabels,omitempty"`
	CommonLabels      map[string]string `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`
}

type Instance struct {
	Spec   InstanceSpec   `json:"spec"`
	Status InstanceStatus `json:"status"`
}

type Operation struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	InstanceID string `json:"instanceId"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// GatewayStatus is the response for GET /v1/instances/{id}/gateway/status.
type GatewayStatus struct {
	InstanceID string `json:"instanceId"`
	Running    bool   `json:"running"`
	Output     string `json:"output,omitempty"`
}

// Error codes for 503 responses — backend should switch on Code, not Error string.
const (
	ErrCodeClusterMaintenance    = "CLUSTER_MAINTENANCE"
	ErrCodeClusterAtCapacity     = "CLUSTER_AT_CAPACITY"
	ErrCodeNodePressure          = "NODE_PRESSURE"
	ErrCodeKubernetesUnreachable = "KUBERNETES_UNREACHABLE"
)

type ErrorResponse struct {
	Error       string `json:"error"`
	Details     string `json:"details,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	// Code is a stable machine-readable identifier for 503 errors so the backend
	// can decide whether to re-route (capacity/maintenance) or retry (transient).
	Code string `json:"code,omitempty"`
}

// ─── Provider config types ────────────────────────────────────────────────────

// ProviderConfigRequest sets an LLM provider's API key for a workspace.
type ProviderConfigRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
}

// SetModelRequest updates the active model, provider, and optional base URL.
type SetModelRequest struct {
	Provider string `json:"provider"`          // openrouter, anthropic, gemini, opencode-go, custom
	Model    string `json:"model"`             // e.g. "anthropic/claude-sonnet-4-6"
	BaseURL  string `json:"baseUrl,omitempty"` // required for "custom" provider
}

// ProviderInfo describes one configured LLM provider for a workspace.
type ProviderInfo struct {
	Provider  string `json:"provider"`
	APIKeySet bool   `json:"apiKeySet"` // true if key exists; the key itself is never returned
	Active    bool   `json:"active"`    // whether this is the currently active provider
}

// ─── Integration types ────────────────────────────────────────────────────────

// IntegrationStatus describes one messaging platform's status for a workspace.
type IntegrationStatus struct {
	Platform string `json:"platform"`
	Enabled  bool   `json:"enabled"`
	// Status is "active", "needs_pairing" (WhatsApp QR not yet scanned), or "disabled".
	Status string `json:"status"`
}

// TelegramIntegrationRequest enables the Telegram messaging integration.
type TelegramIntegrationRequest struct {
	BotToken          string `json:"botToken"`
	WebhookSecret     string `json:"webhookSecret,omitempty"` // required when webhookUrl is set
	AllowedUsers      string `json:"allowedUsers,omitempty"`
	GroupAllowedUsers string `json:"groupAllowedUsers,omitempty"`
	GroupAllowedChats string `json:"groupAllowedChats,omitempty"`
	HomeChannel       string `json:"homeChannel,omitempty"`
	WebhookURL        string `json:"webhookUrl,omitempty"`
	WebhookPort       string `json:"webhookPort,omitempty"`
}

// DiscordIntegrationRequest enables the Discord messaging integration.
// AllowedUsers or AllowedRoles must be set; otherwise all messages are denied.
type DiscordIntegrationRequest struct {
	BotToken             string `json:"botToken"`
	AllowedUsers         string `json:"allowedUsers,omitempty"`
	AllowedRoles         string `json:"allowedRoles,omitempty"`
	HomeChannel          string `json:"homeChannel,omitempty"`
	RequireMention       string `json:"requireMention,omitempty"`
	FreeResponseChannels string `json:"freeResponseChannels,omitempty"`
	IgnoredChannels      string `json:"ignoredChannels,omitempty"`
}

// SlackIntegrationRequest enables the Slack messaging integration.
// AppToken is required for Socket Mode; AllowedUsers should be set.
type SlackIntegrationRequest struct {
	BotToken        string `json:"botToken"`
	AppToken        string `json:"appToken"`
	AllowedUsers    string `json:"allowedUsers,omitempty"`
	AllowedChannels string `json:"allowedChannels,omitempty"`
	HomeChannel     string `json:"homeChannel,omitempty"`
	HomeChannelName string `json:"homeChannelName,omitempty"`
}

// WhatsAppIntegrationRequest enables the WhatsApp integration (Baileys-based).
// After enabling, admin must exec into the pod and run `hermes whatsapp` to pair via QR.
type WhatsAppIntegrationRequest struct {
	AllowedUsers  string `json:"allowedUsers,omitempty"`
	AllowAllUsers bool   `json:"allowAllUsers,omitempty"`
	Mode          string `json:"mode,omitempty"` // defaults to "baileys"
}

// SignalIntegrationRequest enables the Signal messaging integration via signal-cli REST API.
// HTTPURL is the signal-cli REST API endpoint; Account is the registered phone number.
// Signal connection details are stored in k8s Secret (SIGNAL_HTTP_URL, SIGNAL_ACCOUNT).
type SignalIntegrationRequest struct {
	HTTPURL           string `json:"httpUrl"`
	Account           string `json:"account"`
	AllowedUsers      string `json:"allowedUsers,omitempty"`
	GroupAllowedUsers string `json:"groupAllowedUsers,omitempty"`
	HomeChannel       string `json:"homeChannel,omitempty"`
	AllowAllUsers     bool   `json:"allowAllUsers,omitempty"`
	IgnoreStories     bool   `json:"ignoreStories,omitempty"`
}

// DingTalkIntegrationRequest enables the DingTalk messaging integration.
type DingTalkIntegrationRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AllowedUsers string `json:"allowedUsers,omitempty"`
}

// FeishuIntegrationRequest enables the Feishu/Lark messaging integration.
type FeishuIntegrationRequest struct {
	AppID               string `json:"appId"`
	AppSecret           string `json:"appSecret"`
	EncryptKey          string `json:"encryptKey,omitempty"`          // optional — only needed for encrypted events
	VerificationToken   string `json:"verificationToken,omitempty"`   // optional — only needed for event verification
	Domain              string `json:"domain,omitempty"`              // defaults to feishu.cn
	ConnectionMode      string `json:"connectionMode,omitempty"`      // "webhook" or "websocket"
	AllowedUsers        string `json:"allowedUsers,omitempty"`
	HomeChannel         string `json:"homeChannel,omitempty"`
}

// WeComIntegrationRequest enables the WeCom (企业微信) messaging integration.
type WeComIntegrationRequest struct {
	BotID        string `json:"botId"`
	Secret       string `json:"secret"`
	WebsocketURL string `json:"websocketUrl,omitempty"`
	AllowedUsers string `json:"allowedUsers,omitempty"`
	HomeChannel  string `json:"homeChannel,omitempty"`
}

// BlueBubblesIntegrationRequest enables the BlueBubbles (iMessage) integration.
type BlueBubblesIntegrationRequest struct {
	ServerURL    string `json:"serverUrl"`
	Password     string `json:"password"`
	WebhookHost  string `json:"webhookHost,omitempty"`
	WebhookPort  string `json:"webhookPort,omitempty"`
	AllowedUsers string `json:"allowedUsers,omitempty"`
	AllowAllUsers bool  `json:"allowAllUsers,omitempty"`
}

// EmailIntegrationRequest enables the Email messaging integration (IMAP receive + SMTP send).
// Per-instance: each agent profile gets its own email address.
// For Cloudflare Email Routing: point a Cloudflare-routed address to a Gmail/SMTP
// mailbox, then supply those mailbox credentials here.
type EmailIntegrationRequest struct {
	Address      string `json:"address"`               // e.g. agent@yourdomain.com
	Password     string `json:"password"`              // app password or SMTP credential
	IMAPHost     string `json:"imapHost"`              // e.g. imap.gmail.com
	SMTPHost     string `json:"smtpHost"`              // e.g. smtp.gmail.com
	IMAPPort     string `json:"imapPort,omitempty"`    // defaults to 993
	SMTPPort     string `json:"smtpPort,omitempty"`    // defaults to 587
	AllowedUsers string `json:"allowedUsers,omitempty"` // comma-separated sender emails
	HomeAddress  string `json:"homeAddress,omitempty"` // address to use for cron delivery
	PollInterval string `json:"pollInterval,omitempty"` // seconds between IMAP polls (default 15)
	AllowAllUsers bool  `json:"allowAllUsers,omitempty"`
}

// IntegrationsPutRequest is the body for PUT /v1/instances/{id}/integrations.
// Keys are platform names (telegram, discord, slack, whatsapp, signal, dingtalk,
// feishu, wecom, bluebubbles). Platforms absent from the map are disabled.
// An empty map disables all integrations.
type IntegrationsPutRequest map[string]json.RawMessage

// ─── Agent template types ─────────────────────────────────────────────────────

// AgentTemplate is a named snapshot of agent config + SOUL.md content that can
// be applied to any workspace. Stored as a k8s ConfigMap in the bridge namespace.
// Not to be confused with native Hermes profiles (separate HERMES_HOME dirs).
type AgentTemplate struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	// Soul is the content of SOUL.md — the agent's personality/system prompt.
	Soul        string       `json:"soul,omitempty"`
	Config      HermesConfig `json:"config,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// ApplyAgentTemplateRequest applies a named template to a workspace.
type ApplyAgentTemplateRequest struct {
	AgentID string `json:"agentId"`
}
