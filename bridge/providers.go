package main

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// providerSecretKeys maps provider names to the env var / k8s Secret key they inject.
var providerSecretKeys = map[string]string{
	"openrouter":   "OPENROUTER_API_KEY",
	"openai":       "OPENAI_API_KEY",
	"anthropic":    "ANTHROPIC_API_KEY",
	"gemini":       "GOOGLE_API_KEY",
	"opencode-go":  "OPENCODE_GO_API_KEY",
	"opencode-zen": "OPENCODE_ZEN_API_KEY",
	"ai-gateway":   "AI_GATEWAY_API_KEY",
	"deepseek":     "DEEPSEEK_API_KEY",
}

// workspaceSecretName returns the k8s Secret name for a workspace's API keys.
// Matches the chart's runtime-node-core.secretName template when existingSecret is unset.
func (b *Bridge) workspaceSecretName(workspaceID string) string {
	return b.releaseName(workspaceID) + "-secrets"
}

// getOrCreateWorkspaceSecret fetches the workspace k8s Secret, creating it if absent.
func (b *Bridge) getOrCreateWorkspaceSecret(ctx context.Context, ns, secretName string) (*corev1.Secret, error) {
	secret, err := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return secret, nil
	}
	if !k8serrors.IsNotFound(err) {
		return nil, fmt.Errorf("get workspace secret: %w", err)
	}
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
			Labels:    map[string]string{"hermes.ai/managed-by": "bridge"},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}
	created, err := b.KubeClient.CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create workspace secret: %w", err)
	}
	return created, nil
}

// SetWorkspaceProvider stores a provider's API key in the workspace k8s Secret and
// restarts the workspace so the pod picks up the new env var.
func (b *Bridge) SetWorkspaceProvider(ctx context.Context, workspaceID string, req ProviderConfigRequest) error {
	secretKey, ok := providerSecretKeys[req.Provider]
	if !ok {
		return fmt.Errorf("unknown provider %q; valid: %s", req.Provider, strings.Join(providerNames(), ", "))
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	secretName := b.workspaceSecretName(workspaceID)

	secret, err := b.getOrCreateWorkspaceSecret(ctx, ns, secretName)
	if err != nil {
		return err
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[secretKey] = []byte(req.APIKey)
	if _, err := b.KubeClient.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update workspace secret: %w", err)
	}
	b.Logger.Printf("[SetWorkspaceProvider] Updated %s key for workspace %s", req.Provider, workspaceID)
	return b.RestartWorkspace(ctx, workspaceID)
}

// GetWorkspaceProviders lists all providers that have API keys set for the workspace.
// The actual key values are never returned.
func (b *Bridge) GetWorkspaceProviders(ctx context.Context, workspaceID string) ([]ProviderInfo, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	secretName := b.workspaceSecretName(workspaceID)

	secret, err := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, fmt.Errorf("get workspace secret: %w", err)
	}

	// Determine active provider from Helm release config.
	activeProvider := ""
	if cfg, ok := rel.Config["config"].(map[string]any); ok {
		if vals, ok := cfg["values"].(map[string]any); ok {
			if model, ok := vals["model"].(map[string]any); ok {
				activeProvider, _ = model["provider"].(string)
			}
		}
	}

	var infos []ProviderInfo
	for provider, secretKey := range providerSecretKeys {
		apiKeySet := false
		if secret != nil && len(secret.Data[secretKey]) > 0 {
			apiKeySet = true
		}
		if !apiKeySet {
			continue
		}
		infos = append(infos, ProviderInfo{
			Provider:  provider,
			APIKeySet: true,
			Active:    activeProvider == provider,
		})
	}
	return infos, nil
}

// DeleteWorkspaceProvider removes a provider's API key from the workspace k8s Secret
// and restarts the workspace.
func (b *Bridge) DeleteWorkspaceProvider(ctx context.Context, workspaceID, provider string) error {
	secretKey, ok := providerSecretKeys[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	secretName := b.workspaceSecretName(workspaceID)

	secret, err := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get workspace secret: %w", err)
	}
	delete(secret.Data, secretKey)
	if _, err := b.KubeClient.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update workspace secret: %w", err)
	}
	b.Logger.Printf("[DeleteWorkspaceProvider] Removed %s key for workspace %s", provider, workspaceID)
	return b.RestartWorkspace(ctx, workspaceID)
}

// SetWorkspaceModel updates the active model, provider, and optional base URL by
// performing a Helm upgrade that writes a new config.yaml on the next pod start.
func (b *Bridge) SetWorkspaceModel(ctx context.Context, workspaceID string, req SetModelRequest) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.EnvMap = envMapFromRelease(rel.Config)

	if spec.HermesConfig.Model == nil {
		spec.HermesConfig.Model = &ModelConfig{}
	}
	spec.HermesConfig.Model.Default = req.Model
	spec.HermesConfig.Model.Provider = req.Provider
	if req.BaseURL != "" {
		spec.HermesConfig.Model.BaseURL = req.BaseURL
	}
	spec.OverwriteConfig = true

	if _, err := b.UpdateWorkspace(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[SetWorkspaceModel] Updated model=%s provider=%s for workspace %s", req.Model, req.Provider, workspaceID)
	return nil
}

func providerNames() []string {
	names := make([]string, 0, len(providerSecretKeys))
	for k := range providerSecretKeys {
		names = append(names, k)
	}
	return names
}

// envMapFromRelease extracts the extraEnv flat map from the last Helm release config.
// Used to preserve existing custom env vars when doing a partial spec update.
func envMapFromRelease(config map[string]any) map[string]string {
	result := make(map[string]string)
	raw, ok := config["extraEnv"]
	if !ok {
		return result
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return result
	}
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}
