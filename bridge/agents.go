package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	agentTemplateLabelKey   = "hermes.ai/type"
	agentTemplateLabelValue = "agent-template"
	agentTemplateDataKey    = "template.json"
	appliedTemplateAnnotation = "hermes.ai/applied-template"
)

// ListAgentTemplates returns all agent templates stored in the bridge namespace.
func (b *Bridge) ListAgentTemplates(ctx context.Context) ([]AgentTemplate, error) {
	cms, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", agentTemplateLabelKey, agentTemplateLabelValue),
	})
	if err != nil {
		return nil, fmt.Errorf("list agent templates: %w", err)
	}
	templates := make([]AgentTemplate, 0, len(cms.Items))
	for i := range cms.Items {
		t, err := agentTemplateFromConfigMap(&cms.Items[i])
		if err != nil {
			b.Logger.Printf("[ListAgentTemplates] Skipping malformed ConfigMap %s: %v", cms.Items[i].Name, err)
			continue
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// GetAgentTemplate returns a single agent template by ID.
func (b *Bridge) GetAgentTemplate(ctx context.Context, agentID string) (*AgentTemplate, error) {
	cm, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Get(ctx, agentTemplateCMName(agentID), metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, fmt.Errorf("agent template %q not found", agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get agent template: %w", err)
	}
	t, err := agentTemplateFromConfigMap(cm)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateAgentTemplate stores a new agent template as a k8s ConfigMap in the bridge namespace.
func (b *Bridge) CreateAgentTemplate(ctx context.Context, t AgentTemplate) (*AgentTemplate, error) {
	t.ID = "agent-" + randomHex(4)
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	cm, err := agentTemplateToConfigMap(t, b.Config.Namespace)
	if err != nil {
		return nil, err
	}
	created, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create agent template: %w", err)
	}
	result, err := agentTemplateFromConfigMap(created)
	if err != nil {
		return nil, err
	}
	b.Logger.Printf("[CreateAgentTemplate] Created agent template %s (%s)", t.ID, t.Name)
	return &result, nil
}

// UpdateAgentTemplate replaces an existing agent template.
func (b *Bridge) UpdateAgentTemplate(ctx context.Context, agentID string, t AgentTemplate) (*AgentTemplate, error) {
	existing, err := b.GetAgentTemplate(ctx, agentID)
	if err != nil {
		return nil, err
	}
	t.ID = agentID
	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = time.Now().UTC()

	cm, err := agentTemplateToConfigMap(t, b.Config.Namespace)
	if err != nil {
		return nil, err
	}
	// Preserve ResourceVersion for optimistic concurrency.
	existingCM, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Get(ctx, agentTemplateCMName(agentID), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetch existing ConfigMap: %w", err)
	}
	cm.ResourceVersion = existingCM.ResourceVersion

	updated, err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update agent template: %w", err)
	}
	result, err := agentTemplateFromConfigMap(updated)
	if err != nil {
		return nil, err
	}
	b.Logger.Printf("[UpdateAgentTemplate] Updated agent template %s", agentID)
	return &result, nil
}

// DeleteAgentTemplate removes an agent template ConfigMap.
func (b *Bridge) DeleteAgentTemplate(ctx context.Context, agentID string) error {
	err := b.KubeClient.CoreV1().ConfigMaps(b.Config.Namespace).Delete(ctx, agentTemplateCMName(agentID), metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return fmt.Errorf("agent template %q not found", agentID)
	}
	if err != nil {
		return fmt.Errorf("delete agent template: %w", err)
	}
	b.Logger.Printf("[DeleteAgentTemplate] Deleted agent template %s", agentID)
	return nil
}

// ApplyAgentTemplate applies a template's HermesConfig to the workspace via Helm upgrade.
// If the template has a Soul field, it is included in the config values so the chart's
// init container can write it to SOUL.md on next pod start.
func (b *Bridge) ApplyAgentTemplate(ctx context.Context, workspaceID, agentID string) error {
	tmpl, err := b.GetAgentTemplate(ctx, agentID)
	if err != nil {
		return err
	}
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return fmt.Errorf("reconstruct spec: %w", err)
	}
	spec.EnvMap = envMapFromRelease(rel.Config)

	// Merge template config into spec, template fields win.
	spec.HermesConfig = mergeHermesConfig(spec.HermesConfig, tmpl.Config)
	if tmpl.Soul != "" && spec.HermesConfig.Soul == nil {
		spec.HermesConfig.Soul = &SoulConfig{Text: tmpl.Soul}
	} else if tmpl.Soul != "" {
		spec.HermesConfig.Soul.Text = tmpl.Soul
	}
	spec.OverwriteConfig = true

	if _, err := b.UpdateInstance(ctx, spec); err != nil {
		return err
	}

	// Annotate the workspace namespace so GetWorkspaceAgent can identify the applied template.
	ns, err := b.KubeClient.CoreV1().Namespaces().Get(ctx, rel.Namespace, metav1.GetOptions{})
	if err == nil {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		ns.Annotations[appliedTemplateAnnotation] = agentID
		_, _ = b.KubeClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	}

	b.Logger.Printf("[ApplyAgentTemplate] Applied template %s to workspace %s", agentID, workspaceID)
	return b.waitForDeploymentReady(ctx, rel.Namespace, rel.Name, 5*time.Minute)
}

// GetWorkspaceAgent returns the agent template currently applied to the workspace,
// or nil if no template has been applied.
func (b *Bridge) GetInstanceAgent(ctx context.Context, workspaceID string) (*AgentTemplate, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ns, err := b.KubeClient.CoreV1().Namespaces().Get(ctx, rel.Namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get workspace namespace: %w", err)
	}
	agentID := ns.Annotations[appliedTemplateAnnotation]
	if agentID == "" {
		return nil, nil
	}
	return b.GetAgentTemplate(ctx, agentID)
}

// mergeHermesConfig merges override into base; override fields take precedence.
// Only non-nil pointer fields in override replace those in base.
func mergeHermesConfig(base, override HermesConfig) HermesConfig {
	if override.Model != nil {
		base.Model = override.Model
	}
	if override.Agent != nil {
		base.Agent = override.Agent
	}
	if override.Terminal != nil {
		base.Terminal = override.Terminal
	}
	if override.Display != nil {
		base.Display = override.Display
	}
	if override.Browser != nil {
		base.Browser = override.Browser
	}
	if override.Memory != nil {
		base.Memory = override.Memory
	}
	if override.Compression != nil {
		base.Compression = override.Compression
	}
	if override.Security != nil {
		base.Security = override.Security
	}
	if override.Voice != nil {
		base.Voice = override.Voice
	}
	if override.Auxiliary != nil {
		base.Auxiliary = override.Auxiliary
	}
	if override.Gateway != nil {
		base.Gateway = override.Gateway
	}
	if override.Soul != nil {
		base.Soul = override.Soul
	}
	return base
}

func agentTemplateCMName(agentID string) string {
	return "agent-template-" + agentID
}

func agentTemplateToConfigMap(t AgentTemplate, namespace string) (*corev1.ConfigMap, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("marshal agent template: %w", err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentTemplateCMName(t.ID),
			Namespace: namespace,
			Labels: map[string]string{
				agentTemplateLabelKey: agentTemplateLabelValue,
			},
			Annotations: map[string]string{
				"hermes.ai/agent-id":     t.ID,
				"hermes.ai/display-name": t.Name,
				"hermes.ai/created-at":   t.CreatedAt.Format(time.RFC3339),
				"hermes.ai/updated-at":   t.UpdatedAt.Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			agentTemplateDataKey: string(data),
		},
	}, nil
}

func agentTemplateFromConfigMap(cm *corev1.ConfigMap) (AgentTemplate, error) {
	raw, ok := cm.Data[agentTemplateDataKey]
	if !ok {
		return AgentTemplate{}, fmt.Errorf("ConfigMap %s missing %s key", cm.Name, agentTemplateDataKey)
	}
	var t AgentTemplate
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return AgentTemplate{}, fmt.Errorf("unmarshal agent template from %s: %w", cm.Name, err)
	}
	return t, nil
}
