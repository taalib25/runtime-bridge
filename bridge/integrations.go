package main

import (
	"context"
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
	"signal":      {"SIGNAL_HTTP_URL", "SIGNAL_ACCOUNT"},
	"whatsapp":    {}, // no secrets — session is QR-based, stored on PVC
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

	// Reconstruct spec and merge existing + new env vars.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	envMap := envMapFromRelease(rel.Config)
	// Add/update platform-specific env vars from the request.
	for _, key := range platformEnvKeys[platform] {
		if v, ok := cfg[key]; ok && v != "" {
			envMap[key] = v
		}
	}
	spec.EnvMap = envMap
	spec.OverwriteConfig = false

	if _, err := b.UpdateWorkspace(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[EnableIntegration] Enabled %s for workspace %s", platform, workspaceID)
	return b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute)
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

	// Remove platform secret keys.
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

	// Reconstruct spec and remove platform env vars.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	envMap := envMapFromRelease(rel.Config)
	for _, key := range platformEnvKeys[platform] {
		delete(envMap, key)
	}
	spec.EnvMap = envMap
	spec.OverwriteConfig = false

	if _, err := b.UpdateWorkspace(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[DisableIntegration] Disabled %s for workspace %s", platform, workspaceID)
	return b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute)
}

// GetWorkspaceIntegrations returns the status of all known messaging platforms
// for the given workspace. Token values are never returned.
func (b *Bridge) GetWorkspaceIntegrations(ctx context.Context, workspaceID string) ([]IntegrationStatus, error) {
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
		if platform == "whatsapp" {
			enabled = envMap["WHATSAPP_ENABLED"] == "true"
		} else {
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
