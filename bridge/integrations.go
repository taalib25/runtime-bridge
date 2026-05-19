package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// platformSecretKeys lists the k8s Secret keys required by each messaging platform.
var platformSecretKeys = map[string][]string{
	"telegram":    {"TELEGRAM_BOT_TOKEN", "TELEGRAM_WEBHOOK_SECRET"},
	"discord":     {"DISCORD_BOT_TOKEN"},
	"slack":       {"SLACK_BOT_TOKEN", "SLACK_APP_TOKEN"},
	"signal":      {"SIGNAL_HTTP_URL", "SIGNAL_ACCOUNT"}, // stored in k8s Secret (chart treats them as secrets)
	"whatsapp":    {},                   // no secrets — session is QR-based, stored on PVC
	"dingtalk":    {"DINGTALK_CLIENT_ID", "DINGTALK_CLIENT_SECRET"},
	"feishu":      {"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_ENCRYPT_KEY", "FEISHU_VERIFICATION_TOKEN"},
	"wecom":       {"WECOM_BOT_ID", "WECOM_SECRET"},
	"bluebubbles": {"BLUEBUBBLES_SERVER_URL", "BLUEBUBBLES_PASSWORD"},
}

// platformEnvKeys lists the plain env var keys for each platform's config.
var platformEnvKeys = map[string][]string{
	"telegram": {
		"TELEGRAM_ALLOWED_USERS", "TELEGRAM_GROUP_ALLOWED_USERS",
		"TELEGRAM_GROUP_ALLOWED_CHATS", "TELEGRAM_HOME_CHANNEL",
		"TELEGRAM_HOME_CHANNEL_NAME", "TELEGRAM_WEBHOOK_URL", "TELEGRAM_WEBHOOK_PORT",
	},
	"discord": {
		"DISCORD_ALLOWED_USERS", "DISCORD_ALLOWED_ROLES", "DISCORD_ALLOWED_CHANNELS",
		"DISCORD_HOME_CHANNEL", "DISCORD_HOME_CHANNEL_NAME", "DISCORD_REQUIRE_MENTION",
		"DISCORD_FREE_RESPONSE_CHANNELS", "DISCORD_IGNORED_CHANNELS",
	},
	"slack":    {"SLACK_ALLOWED_USERS", "SLACK_HOME_CHANNEL", "SLACK_HOME_CHANNEL_NAME"},
	"whatsapp": {"WHATSAPP_ENABLED", "WHATSAPP_MODE", "WHATSAPP_ALLOWED_USERS", "WHATSAPP_ALLOW_ALL_USERS"},
	"signal": {
		"SIGNAL_ALLOWED_USERS", "SIGNAL_GROUP_ALLOWED_USERS",
		"SIGNAL_HOME_CHANNEL_NAME", "SIGNAL_ALLOW_ALL_USERS", "SIGNAL_IGNORE_STORIES",
	},
	"dingtalk":    {"DINGTALK_ALLOWED_USERS"},
	"feishu":      {"FEISHU_DOMAIN", "FEISHU_CONNECTION_MODE", "FEISHU_ALLOWED_USERS", "FEISHU_HOME_CHANNEL"},
	"wecom":       {"WECOM_WEBSOCKET_URL", "WECOM_ALLOWED_USERS", "WECOM_HOME_CHANNEL"},
	"bluebubbles": {"BLUEBUBBLES_WEBHOOK_HOST", "BLUEBUBBLES_WEBHOOK_PORT", "BLUEBUBBLES_ALLOWED_USERS", "BLUEBUBBLES_ALLOW_ALL_USERS"},
}

// platformAllowedUsersKey maps each platform to its ALLOWED_USERS env var.
// Used to decide whether to set GATEWAY_ALLOW_ALL_USERS as the default.
var platformAllowedUsersKey = map[string]string{
	"telegram":    "TELEGRAM_ALLOWED_USERS",
	"discord":     "DISCORD_ALLOWED_USERS",
	"slack":       "SLACK_ALLOWED_USERS",
	"whatsapp":    "WHATSAPP_ALLOWED_USERS",
	"signal":      "SIGNAL_ALLOWED_USERS",
	"dingtalk":    "DINGTALK_ALLOWED_USERS",
	"feishu":      "FEISHU_ALLOWED_USERS",
	"wecom":       "WECOM_ALLOWED_USERS",
	"bluebubbles": "BLUEBUBBLES_ALLOWED_USERS",
}

// applyGatewayAllowAll sets GATEWAY_ALLOW_ALL_USERS=true in envMap when no
// platform-specific allowedUsers is configured, ensuring users aren't locked
// out by default. Clears it when any allowedUsers is present (caller is
// managing access explicitly).
func applyGatewayAllowAll(envMap map[string]string) {
	for _, key := range platformAllowedUsersKey {
		if envMap[key] != "" {
			delete(envMap, "GATEWAY_ALLOW_ALL_USERS")
			return
		}
	}
	envMap["GATEWAY_ALLOW_ALL_USERS"] = "true"
}

// EnableIntegration stores platform tokens in the workspace k8s Secret and
// updates the workspace's env vars so the gateway picks them up on next start.
// cfg is a flat map of secret-key→value and env-key→value pairs for the platform.
func (b *Bridge) EnableIntegration(ctx context.Context, workspaceID, platform string, cfg map[string]string) error {
	if _, ok := platformSecretKeys[platform]; !ok {
		return fmt.Errorf("unsupported platform %q", platform)
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
	secretName := b.workspaceSecretName(workspaceID)

	// Patch k8s Secret with token keys.
	if len(platformSecretKeys[platform]) > 0 {
		secret, err := b.getOrCreateWorkspaceSecret(ctx, ns, secretName)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for _, key := range platformSecretKeys[platform] {
			if v, ok := cfg[key]; ok && v != "" {
				secret.Data[key] = []byte(v)
			}
		}
		if _, err := b.KubeClient.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update workspace secret: %w", err)
		}
	}

	// Reconstruct spec; workspaceSpecFromRelease restores spec.Secrets via secretKeysFromRelease.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	envMap := envMapFromRelease(rel.Config)
	for _, key := range platformEnvKeys[platform] {
		if v, ok := cfg[key]; ok && v != "" {
			envMap[key] = v
		}
	}
	applyGatewayAllowAll(envMap)
	spec.EnvMap = envMap
	// Register platform secret keys so the Deployment gets secretKeyRef entries.
	for _, key := range platformSecretKeys[platform] {
		spec.Secrets[key] = ""
	}
	spec.OverwriteConfig = false

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[EnableIntegration] Enabled %s for instance %s", platform, workspaceID)
	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute); err != nil {
		return err
	}
	b.EnsureGatewayRunning(ctx, workspaceID)
	return nil
}

// DisableIntegration removes platform tokens from the workspace k8s Secret and
// clears all platform env vars from the workspace's extraEnv.
func (b *Bridge) DisableIntegration(ctx context.Context, workspaceID, platform string) error {
	if _, ok := platformSecretKeys[platform]; !ok {
		return fmt.Errorf("unsupported platform %q", platform)
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
	secretName := b.workspaceSecretName(workspaceID)

	// Remove platform secret keys from bridge-owned Secret.
	secret, err := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("get workspace secret: %w", err)
	}
	if err == nil && secret.Data != nil {
		for _, key := range platformSecretKeys[platform] {
			delete(secret.Data, key)
		}
		if _, err := b.KubeClient.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update workspace secret: %w", err)
		}
	}

	// Reconstruct spec; spec.Secrets is restored by workspaceSpecFromRelease.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	envMap := envMapFromRelease(rel.Config)
	for _, key := range platformEnvKeys[platform] {
		delete(envMap, key)
	}
	spec.EnvMap = envMap
	// Remove platform secret keys from extraSecretKeys so the Deployment drops the refs.
	for _, key := range platformSecretKeys[platform] {
		delete(spec.Secrets, key)
	}
	spec.OverwriteConfig = false

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[DisableIntegration] Disabled %s for instance %s", platform, workspaceID)
	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute); err != nil {
		return err
	}
	b.EnsureGatewayRunning(ctx, workspaceID)
	return nil
}

// GetWorkspaceIntegrations returns the status of all known messaging platforms
// for the given workspace. Token values are never returned.
func (b *Bridge) GetInstanceIntegrations(ctx context.Context, workspaceID string) ([]IntegrationStatus, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	secretName := b.workspaceSecretName(workspaceID)

	secret, _ := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	envMap := envMapFromRelease(rel.Config)

	var statuses []IntegrationStatus
	for platform, secretKeys := range platformSecretKeys {
		enabled := false
		switch platform {
		case "whatsapp":
			enabled = envMap["WHATSAPP_ENABLED"] == "true"
		default:
			for _, key := range secretKeys {
				if secret != nil && len(secret.Data[key]) > 0 {
					enabled = true
					break
				}
			}
		}

		status := "disabled"
		if enabled {
			status = "active"
			if platform == "whatsapp" {
				status = "needs_pairing"
			}
		}
		statuses = append(statuses, IntegrationStatus{
			Platform: platform,
			Enabled:  enabled,
			Status:   status,
		})
	}
	return statuses, nil
}

// SetIntegrations converges the instance to exactly the set of messaging platforms
// in desired. Platforms absent from the map are disabled; an empty map disables all.
// All changes are applied in a single Helm upgrade — not one upgrade per platform.
func (b *Bridge) SetIntegrations(ctx context.Context, workspaceID string, desired map[string]json.RawMessage) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
	secretName := b.workspaceSecretName(workspaceID)

	// Build union sets of all known platform keys so we can strip them cleanly.
	allSecretKeys := map[string]bool{}
	allEnvKeys := map[string]bool{}
	for _, keys := range platformSecretKeys {
		for _, k := range keys {
			allSecretKeys[k] = true
		}
	}
	for _, keys := range platformEnvKeys {
		for _, k := range keys {
			allEnvKeys[k] = true
		}
	}

	// Get current secret — create if missing.
	secret, err := b.getOrCreateWorkspaceSecret(ctx, ns, secretName)
	if err != nil {
		return err
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	// Strip all platform secret keys from current secret data.
	for k := range allSecretKeys {
		delete(secret.Data, k)
	}

	// Strip all platform env keys from current envMap.
	envMap := envMapFromRelease(rel.Config)
	for k := range allEnvKeys {
		delete(envMap, k)
	}

	// Apply desired platforms.
	for platform, raw := range desired {
		cfg, err := decodeIntegrationCfg(platform, raw)
		if err != nil {
			return fmt.Errorf("platform %s: %w", platform, err)
		}
		// Split cfg into secrets and plain env vars.
		secretKeySet := map[string]bool{}
		for _, k := range platformSecretKeys[platform] {
			secretKeySet[k] = true
		}
		for k, v := range cfg {
			if secretKeySet[k] {
				secret.Data[k] = []byte(v)
			} else {
				envMap[k] = v
			}
		}
	}

	// Persist updated k8s Secret.
	if _, err := b.KubeClient.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update workspace secret: %w", err)
	}

	// Reconstruct spec and apply new env + secret key registrations.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	applyGatewayAllowAll(envMap)
	spec.EnvMap = envMap

	// Rebuild spec.Secrets to reflect what is actually in the k8s Secret.
	// Preserve non-platform keys (e.g. API_SERVER_KEY, provider API keys).
	for k := range allSecretKeys {
		delete(spec.Secrets, k)
	}
	for k := range secret.Data {
		if allSecretKeys[k] {
			spec.Secrets[k] = "" // key name only — value lives in k8s Secret
		}
	}

	spec.OverwriteConfig = false

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[SetIntegrations] Converged %d platform(s) for instance %s", len(desired), workspaceID)
	if err := b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute); err != nil {
		return err
	}
	b.EnsureGatewayRunning(ctx, workspaceID)
	return nil
}
