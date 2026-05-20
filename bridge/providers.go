package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// providerSecretKeys maps provider names to the env var / k8s Secret key they inject.
// Sourced from the hermes-agent chart's supported provider list.
var providerSecretKeys = map[string]string{
	// Core western providers
	"openai":     "OPENAI_API_KEY",
	"anthropic":  "ANTHROPIC_API_KEY",
	"openrouter": "OPENROUTER_API_KEY",
	"gemini":     "GOOGLE_API_KEY",
	"groq":       "GROQ_API_KEY",
	"mistral":    "MISTRAL_API_KEY",
	// Nous Research
	"nous": "NOUS_API_KEY",
	// OpenCode inference tiers
	"opencode-go":  "OPENCODE_GO_API_KEY",
	"opencode-zen": "OPENCODE_ZEN_API_KEY",
	// Chinese / Asia-Pacific providers
	"glm":        "GLM_API_KEY",
	"minimax":    "MINIMAX_API_KEY",
	"kimi":       "KIMI_API_KEY",
	"huggingface": "HF_TOKEN",
	// AI gateway / proxy
	"ai-gateway": "AI_GATEWAY_API_KEY",
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
// performs a Helm upgrade so the Deployment gets the secretKeyRef for the key, then
// waits for the rolling update to complete.
func (b *Bridge) SetInstanceProvider(ctx context.Context, workspaceID string, req ProviderConfigRequest) error {
	secretKey, ok := providerSecretKeys[req.Provider]
	if !ok {
		return fmt.Errorf("unknown provider %q; valid: %s", req.Provider, strings.Join(providerNames(), ", "))
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
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

	// Helm upgrade: register the key in extraSecretKeys so the Deployment references it.
	// workspaceSpecFromRelease already restores existing secret keys via secretKeysFromRelease.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.EnvMap = envMapFromRelease(rel.Config)
	spec.Secrets[secretKey] = "" // value lives in bridge-owned Secret
	spec.OverwriteConfig = false

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[SetInstanceProvider] Updated %s key for instance %s", req.Provider, workspaceID)
	return b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute)
}

// GetWorkspaceProviders lists all providers that have API keys set for the workspace.
// The actual key values are never returned.
func (b *Bridge) GetInstanceProviders(ctx context.Context, workspaceID string) ([]ProviderInfo, error) {
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
// and performs a Helm upgrade to remove the secretKeyRef from the Deployment.
func (b *Bridge) DeleteInstanceProvider(ctx context.Context, workspaceID, provider string) error {
	secretKey, ok := providerSecretKeys[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
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

	// Helm upgrade: remove the key from extraSecretKeys so the Deployment no longer references it.
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.EnvMap = envMapFromRelease(rel.Config)
	delete(spec.Secrets, secretKey)
	spec.OverwriteConfig = false

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[DeleteInstanceProvider] Removed %s key for instance %s", provider, workspaceID)
	return b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute)
}

// SetWorkspaceModel updates the active model, provider, and optional base URL by
// performing a Helm upgrade that writes a new config.yaml on the next pod start.
func (b *Bridge) SetInstanceModel(ctx context.Context, workspaceID string, req SetModelRequest) error {
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

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[SetInstanceModel] Updated model=%s provider=%s for instance %s", req.Model, req.Provider, workspaceID)
	return nil
}

// GetInstanceConfig returns the HermesConfig stored in the last Helm release.
func (b *Bridge) GetInstanceConfig(ctx context.Context, workspaceID string) (HermesConfig, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return HermesConfig{}, err
	}
	return hermesConfigFromRelease(rel.Config), nil
}

// SetInstanceConfig merges the caller-supplied HermesConfig fields into the
// instance's stored config and triggers a Helm upgrade so the next pod start
// picks up the new config.yaml.
func (b *Bridge) SetInstanceConfig(ctx context.Context, workspaceID string, incoming HermesConfig) error {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	ns := rel.Namespace
	releaseName := rel.Name
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, ns, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.EnvMap = envMapFromRelease(rel.Config)

	// Merge: incoming fields override stored fields; nil sections are left as-is.
	existing := spec.HermesConfig
	if incoming.Model != nil {
		existing.Model = incoming.Model
	}
	if incoming.Agent != nil {
		existing.Agent = incoming.Agent
	}
	if incoming.Terminal != nil {
		existing.Terminal = incoming.Terminal
	}
	if incoming.Display != nil {
		existing.Display = incoming.Display
	}
	if incoming.Browser != nil {
		existing.Browser = incoming.Browser
	}
	if incoming.Memory != nil {
		existing.Memory = incoming.Memory
	}
	if incoming.Compression != nil {
		existing.Compression = incoming.Compression
	}
	if incoming.Security != nil {
		existing.Security = incoming.Security
	}
	if incoming.Voice != nil {
		existing.Voice = incoming.Voice
	}
	if incoming.Auxiliary != nil {
		existing.Auxiliary = incoming.Auxiliary
	}
	if incoming.Gateway != nil {
		existing.Gateway = incoming.Gateway
	}
	if incoming.Session != nil {
		existing.Session = incoming.Session
	}
	if incoming.Soul != nil {
		existing.Soul = incoming.Soul
	}
	spec.HermesConfig = existing
	spec.OverwriteConfig = true

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}
	b.Logger.Printf("[SetInstanceConfig] Updated config for instance %s", workspaceID)
	return b.waitForDeploymentReady(ctx, ns, releaseName, 3*time.Minute)
}

func providerNames() []string {
	names := make([]string, 0, len(providerSecretKeys))
	for k := range providerSecretKeys {
		names = append(names, k)
	}
	return names
}

// secretKeysFromRelease reads the extraSecretKeys map stored in the last Helm release
// config and returns it as spec.Secrets key names (values are empty — real values live
// in the bridge-owned k8s Secret). Used to preserve existing secretKeyRef entries on
// partial spec updates.
func secretKeysFromRelease(config map[string]any) map[string]string {
	result := make(map[string]string)
	raw, ok := config["extraSecretKeys"]
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
