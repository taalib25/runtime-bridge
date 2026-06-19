package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

type Config struct {
	ListenAddress     string        `json:"listenAddress" yaml:"listenAddress"`
	Namespace         string        `json:"namespace" yaml:"namespace"`
	ClusterName       string        `json:"clusterName" yaml:"clusterName"`
	KubeconfigPath    string        `json:"kubeconfigPath" yaml:"kubeconfigPath"`
	ChartPath         string        `json:"chartPath" yaml:"chartPath"`
	BridgeSecret      string        `json:"bridgeSecret" yaml:"bridgeSecret"`
	SyncInterval      time.Duration `json:"syncInterval" yaml:"syncInterval"`
	ShutdownTimeout   time.Duration `json:"shutdownTimeout" yaml:"shutdownTimeout"`
	HTTPClientTimeout time.Duration `json:"httpClientTimeout" yaml:"httpClientTimeout"`
	OperationTimeout  time.Duration `json:"operationTimeout" yaml:"operationTimeout"`
	HealthPath        string        `json:"healthPath" yaml:"healthPath"`
	ReleasePrefix     string        `json:"releasePrefix" yaml:"releasePrefix"`
	CreateNamespace   bool          `json:"createNamespace" yaml:"createNamespace"`
	// DefaultForwardAuthURL is configured (env: BRIDGE_FORWARD_AUTH_URL) but currently DEAD:
	// nothing applies it to spec.ForwardAuthURL when a create request omits the field, so
	// ForwardAuth only ever activates if the caller sets forwardAuthURL explicitly (apps/server
	// doesn't, as of 2026-06-19). TODO: either wire this default in, or remove it.
	DefaultForwardAuthURL string `json:"defaultForwardAuthURL" yaml:"defaultForwardAuthURL"`
	// DefaultCORSOrigins is a comma-separated list of allowed browser origins applied to every
	// workspace that doesn't set corsOrigins explicitly.
	// e.g. "https://app.hermeshq.net,http://localhost:3002"
	// Env: BRIDGE_CORS_ORIGINS
	DefaultCORSOrigins string `json:"defaultCORSOrigins" yaml:"defaultCORSOrigins"`
	// DefaultDomain is appended to instanceID to form the ingress host when network.host is unset.
	// e.g. "hermeshq.net" → "tenant-abc.hermeshq.net"
	// Env: BRIDGE_DEFAULT_DOMAIN
	DefaultDomain string `json:"defaultDomain" yaml:"defaultDomain"`

	// RuntimeNodeCoreImage is the default image repository used when runtimeMode=runtime-node-core
	// and no explicit image is provided in the workspace spec.
	// Env: BRIDGE_RUNTIME_NODE_CORE_IMAGE
	RuntimeNodeCoreImage string `json:"runtimeNodeCoreImage" yaml:"runtimeNodeCoreImage"`
	// RuntimeNodeCoreImageTag is the default image tag for runtimeMode=runtime-node-core.
	// Env: BRIDGE_RUNTIME_NODE_CORE_TAG
	RuntimeNodeCoreImageTag string `json:"runtimeNodeCoreImageTag" yaml:"runtimeNodeCoreImageTag"`
	// RuntimeNodeCoreImageDigest pins the runtime image by content digest (sha256:...)
	// instead of by tag, for reproducibility and so IfNotPresent caching/pre-pulling
	// actually hits — a floating "latest" tag means a fresh push upstream invalidates
	// every node's cache silently. When set, this takes precedence over
	// RuntimeNodeCoreImageTag in the Helm values passed to the chart.
	// Env: BRIDGE_RUNTIME_NODE_CORE_DIGEST
	RuntimeNodeCoreImageDigest string `json:"runtimeNodeCoreImageDigest" yaml:"runtimeNodeCoreImageDigest"`
	ConfigFile string `json:"-" yaml:"-"`
}

func DefaultConfig() Config {
	return Config{
		ListenAddress:           ":8080",
		Namespace:               "default",
		SyncInterval:            5 * time.Minute,
		ShutdownTimeout:         10 * time.Second,
		HTTPClientTimeout:       5 * time.Second,
		OperationTimeout:        10 * time.Minute,
		// The Hermes dashboard's liveness endpoint (HERMES_DASHBOARD_INSECURE=1 means the
		// whole dashboard is unauthenticated anyway, but this stays the canonical health
		// path regardless). (Plain /health 401s/404s → false unhealthy → would break
		// status "ready" + auto-rollback restart/upgrade/repair/redeploy.)
		HealthPath:              "/api/status",
		ReleasePrefix:           "",
		CreateNamespace:         false,
		RuntimeNodeCoreImage: "nousresearch/hermes-agent",
		// Official upstream image. It ships the web dashboard as the `hermes dashboard`
		// subcommand (port 9119), runs as root under tini (no s6), HERMES_HOME=/opt/data.
		RuntimeNodeCoreImageTag: "latest",
		// Pinned to what `nousresearch/hermes-agent:latest` resolved to as of 2026-06-18
		// (resolved via the Docker Hub registry API). Upstream publishes no versioned
		// tags, so this is the only way to get reproducibility + working IfNotPresent
		// caching/pre-pulling. Re-resolve and bump deliberately when upstream updates —
		// never let this silently drift by removing it.
		RuntimeNodeCoreImageDigest: "sha256:be6912a818cb4a479fa384024a599fee6796d56d1e6a5ca56834deec328d9981",
	}
}

func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	cfg.ConfigFile = strings.TrimSpace(os.Getenv("BRIDGE_CONFIG_FILE"))

	if cfg.ConfigFile != "" {
		payload, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(payload, &cfg); err != nil {
			return Config{}, fmt.Errorf("decode config file: %w", err)
		}
	}

	overlayString(&cfg.ListenAddress, "BRIDGE_LISTEN_ADDRESS")
	overlayString(&cfg.Namespace, "BRIDGE_NAMESPACE")
	overlayString(&cfg.ClusterName, "BRIDGE_CLUSTER_NAME")
	overlayString(&cfg.KubeconfigPath, "BRIDGE_KUBECONFIG")
	overlayString(&cfg.ChartPath, "BRIDGE_CHART_PATH")
	overlayString(&cfg.BridgeSecret, "BRIDGE_SECRET")
	overlayString(&cfg.HealthPath, "BRIDGE_HEALTH_PATH")
	overlayString(&cfg.ReleasePrefix, "BRIDGE_RELEASE_PREFIX")
	overlayString(&cfg.DefaultForwardAuthURL, "BRIDGE_FORWARD_AUTH_URL")
	overlayString(&cfg.DefaultCORSOrigins, "BRIDGE_CORS_ORIGINS")
	overlayString(&cfg.DefaultDomain, "BRIDGE_DEFAULT_DOMAIN")
	overlayString(&cfg.RuntimeNodeCoreImage, "BRIDGE_RUNTIME_NODE_CORE_IMAGE")
	overlayString(&cfg.RuntimeNodeCoreImageTag, "BRIDGE_RUNTIME_NODE_CORE_TAG")
	overlayString(&cfg.RuntimeNodeCoreImageDigest, "BRIDGE_RUNTIME_NODE_CORE_DIGEST")

	if err := overlayDuration(&cfg.SyncInterval, "BRIDGE_SYNC_INTERVAL"); err != nil {
		return Config{}, err
	}
	if err := overlayDuration(&cfg.ShutdownTimeout, "BRIDGE_SHUTDOWN_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if err := overlayDuration(&cfg.HTTPClientTimeout, "BRIDGE_HTTP_CLIENT_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if err := overlayDuration(&cfg.OperationTimeout, "BRIDGE_OPERATION_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if err := overlayBool(&cfg.CreateNamespace, "BRIDGE_CREATE_NAMESPACE"); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ClusterName) == "" {
		return fmt.Errorf("clusterName is required")
	}
	if strings.TrimSpace(c.ChartPath) == "" {
		return fmt.Errorf("chartPath is required")
	}
	if strings.TrimSpace(c.BridgeSecret) == "" {
		return fmt.Errorf("bridgeSecret is required")
	}
	if strings.TrimSpace(c.ListenAddress) == "" {
		return fmt.Errorf("listenAddress is required")
	}
	if strings.TrimSpace(c.Namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if c.SyncInterval <= 0 {
		return fmt.Errorf("syncInterval must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdownTimeout must be positive")
	}
	if c.HTTPClientTimeout <= 0 {
		return fmt.Errorf("httpClientTimeout must be positive")
	}
	if c.OperationTimeout <= 0 {
		return fmt.Errorf("operationTimeout must be positive")
	}
	return nil
}

func overlayString(target *string, envKey string) {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		*target = value
	}
}

func overlayDuration(target *time.Duration, envKey string) error {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", envKey, err)
	}
	*target = parsed
	return nil
}

func overlayBool(target *bool, envKey string) error {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", envKey, err)
	}
	*target = parsed
	return nil
}
