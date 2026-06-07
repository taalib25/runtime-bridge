package main

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Label & annotation contract — see docs/label-contract.md.
//
// Backend owns business metadata under the hermescloud.dev/* prefix only.
// The bridge owns Kubernetes identity and standard labels, validates backend
// metadata, and never mutates it.

const (
	// backendLabelPrefix is the only key prefix the backend may use for
	// commonLabels / commonAnnotations. Everything else is rejected.
	backendLabelPrefix = "hermescloud.dev/"

	// chartName is the fixed app.kubernetes.io/name for every instance — it MUST match
	// the Helm chart's Chart.Name (charts/hermes-agent) so the bridge's derived selector
	// labels equal the chart-rendered pod selector.
	chartName = "hermes-agent"

	// namespaceManagedByLabel is the bridge's own operational anchor on instance
	// namespaces. Control operations (drain, cluster summary) select on it. It is
	// NOT business metadata, so it does not violate "bridge invents no business facts".
	namespaceManagedByKey   = "hermeshq/managed-by"
	namespaceManagedByValue = "bridge"
)

// reservedLabelPrefixes are key prefixes the backend may never set. The
// hermescloud.dev/* allowlist already excludes these, but they are listed
// explicitly so validation errors can name the conflict precisely.
var reservedLabelPrefixes = []string{
	"app.kubernetes.io/",
	"helm.sh/",
	"meta.helm.sh/",
	"kubernetes.io/",
	"k8s.io/",
	"hermeshq/",
}

// deriveSelectorLabels returns the immutable Kubernetes identity for an instance.
// It is a pure function of the release name, so it is identical on every
// create/upgrade/repair — there is nothing for the backend to supply or drift.
func deriveSelectorLabels(releaseName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     chartName,
		"app.kubernetes.io/instance": releaseName,
	}
}

// instanceLabels builds the label set for a bridge-created (imperative) resource:
// derived identity + standard labels + backend commonLabels. Used for the namespace,
// IngressRoute, and bridge-owned Secret. Chart-rendered resources get their labels
// from the chart helper instead; this mirrors that set for imperative resources.
//
// Chart-owned keys are written last so they win over any (already-rejected) backend
// collision — belt and suspenders on top of validation.
func (b *Bridge) instanceLabels(spec InstanceSpec) map[string]string {
	out := map[string]string{}
	for k, v := range spec.CommonLabels {
		out[k] = v
	}
	for k, v := range deriveSelectorLabels(b.releaseName(spec.InstanceID)) {
		out[k] = v
	}
	out["app.kubernetes.io/part-of"] = chartName
	// Imperative resources are managed by the bridge, not Helm — set truthfully.
	out["app.kubernetes.io/managed-by"] = "hermes-bridge"
	return out
}

// instanceAnnotations returns the backend commonAnnotations for imperative resources.
// Nil/empty is returned as nil so callers can skip setting an annotations block.
func (b *Bridge) instanceAnnotations(spec InstanceSpec) map[string]string {
	if len(spec.CommonAnnotations) == 0 {
		return nil
	}
	out := make(map[string]string, len(spec.CommonAnnotations))
	for k, v := range spec.CommonAnnotations {
		out[k] = v
	}
	return out
}

// validateBackendMetadata enforces the label/annotation contract on backend input.
// It rejects (never mutates) and returns an error suitable for a 422 response.
// Empty maps are valid.
func validateBackendMetadata(spec InstanceSpec) error {
	for key, value := range spec.CommonLabels {
		if err := validateBackendKey("commonLabels", key); err != nil {
			return err
		}
		if msgs := validation.IsValidLabelValue(value); len(msgs) > 0 {
			return fmt.Errorf("commonLabels[%q] has an invalid label value (%s); move long values to commonAnnotations", key, strings.Join(msgs, "; "))
		}
	}
	for key := range spec.CommonAnnotations {
		if err := validateBackendKey("commonAnnotations", key); err != nil {
			return err
		}
		// Annotation values are intentionally unconstrained (long strings, URLs,
		// JSON allowed). The k8s total-size budget is enforced by the API server.
	}
	return nil
}

func validateBackendKey(field, key string) error {
	if !strings.HasPrefix(key, backendLabelPrefix) {
		for _, p := range reservedLabelPrefixes {
			if strings.HasPrefix(key, p) {
				return fmt.Errorf("%s key %q uses reserved prefix %q; backend may only use the %q prefix", field, key, p, backendLabelPrefix)
			}
		}
		return fmt.Errorf("%s key %q is not allowed; backend may only use the %q prefix", field, key, backendLabelPrefix)
	}
	if msgs := validation.IsQualifiedName(key); len(msgs) > 0 {
		return fmt.Errorf("%s key %q is not a valid Kubernetes label key (%s)", field, key, strings.Join(msgs, "; "))
	}
	return nil
}
