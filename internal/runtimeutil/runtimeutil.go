package runtimeutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	runtimev1alpha1 "github.com/hermeshq/hermes-runtime-operator/api/v1alpha1"
)

func Labels(rt *runtimev1alpha1.Runtime) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":         "runtime",
		"app.kubernetes.io/managed-by":   "hermes-runtime-operator",
		"app.kubernetes.io/instance":     rt.Spec.RuntimeID,
		"hermes.hermeshq.net/runtime-id": rt.Spec.RuntimeID,
		"hermes.hermeshq.net/tenant-id":  rt.Spec.TenantID,
	}
}

func DeploymentName(runtimeID string) string {
	return runtimeID
}

func ServiceName(runtimeID string) string {
	return runtimeID
}

func ConfigMapName(runtimeID string) string {
	return fmt.Sprintf("%s-config", runtimeID)
}

func PVCName(runtimeID string) string {
	return fmt.Sprintf("%s-data", runtimeID)
}

func IngressName(runtimeID string) string {
	return fmt.Sprintf("%s-ingress", runtimeID)
}

func ManagedSecretName(runtimeID string) string {
	return fmt.Sprintf("%s-secrets", runtimeID)

}

func Host(spec runtimev1alpha1.RuntimeSpec) (string, error) {
	switch spec.Network.Mode {
	case runtimev1alpha1.RuntimeNetworkModeSubdomain:
		if strings.TrimSpace(spec.Network.Subdomain) == "" {
			return "", fmt.Errorf("network.subdomain is required for subdomain mode")
		}
		return fmt.Sprintf("%s.%s", spec.Network.Subdomain, spec.Network.Host), nil
	case runtimev1alpha1.RuntimeNetworkModePath:
		return spec.Network.Host, nil
	default:
		return "", fmt.Errorf("unsupported network mode %q", spec.Network.Mode)
	}
}

func Path(spec runtimev1alpha1.RuntimeSpec) (string, error) {
	switch spec.Network.Mode {
	case runtimev1alpha1.RuntimeNetworkModeSubdomain:
		return "/", nil
	case runtimev1alpha1.RuntimeNetworkModePath:
		if strings.TrimSpace(spec.Network.Path) == "" {
			return "", fmt.Errorf("network.path is required for path mode")
		}
		if strings.HasPrefix(spec.Network.Path, "/") {
			return spec.Network.Path, nil
		}
		return "/" + spec.Network.Path, nil
	default:
		return "", fmt.Errorf("unsupported network mode %q", spec.Network.Mode)
	}
}

func URL(spec runtimev1alpha1.RuntimeSpec) (string, error) {
	host, err := Host(spec)
	if err != nil {
		return "", err
	}
	path, err := Path(spec)
	if err != nil {
		return "", err
	}
	if path == "/" {
		return "https://" + host, nil
	}
	return "https://" + host + path, nil
}

func RenderConfigYAML(spec runtimev1alpha1.RuntimeSpec) (string, error) {
	if len(spec.Config.Raw) == 0 {
		return "{}\n", nil
	}
	configYAML, err := yaml.JSONToYAML(spec.Config.Raw)
	if err != nil {
		return "", err
	}
	if len(configYAML) == 0 {
		return "{}\n", nil
	}
	if configYAML[len(configYAML)-1] != '\n' {
		configYAML = append(configYAML, '\n')
	}
	return string(configYAML), nil
}

func ConfigChecksum(configYAML string, soul string) string {
	hash := sha256.Sum256([]byte(configYAML + "\n---\n" + soul))
	return hex.EncodeToString(hash[:])
}

func SecretChecksum(secret *corev1.Secret) string {
	if secret == nil || len(secret.Data) == 0 {
		return ""
	}
	keys := make([]string, 0, len(secret.Data))
	for key := range secret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b := strings.Builder{}
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString("=")
		b.Write(secret.Data[key])
		b.WriteString("\n")
	}
	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

func ContainsSecretRef(secretRefs []runtimev1alpha1.RuntimeSecretReference, name string) bool {
	for _, item := range secretRefs {
		if item.Name == name {
			return true
		}
	}
	return false
}

func WithSecretRef(secretRefs []runtimev1alpha1.RuntimeSecretReference, name string) []runtimev1alpha1.RuntimeSecretReference {
	if ContainsSecretRef(secretRefs, name) {
		return secretRefs
	}
	return append(secretRefs, runtimev1alpha1.RuntimeSecretReference{Name: name})
}

func WithoutSecretRef(secretRefs []runtimev1alpha1.RuntimeSecretReference, name string) []runtimev1alpha1.RuntimeSecretReference {
	result := make([]runtimev1alpha1.RuntimeSecretReference, 0, len(secretRefs))
	for _, item := range secretRefs {
		if item.Name != name {
			result = append(result, item)
		}
	}
	return result
}
