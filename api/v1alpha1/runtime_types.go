/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RuntimePlan defines resource limits for the runtime workload.
type RuntimePlan struct {
	// CPUMillis is the CPU limit in milli-cores (e.g., 1000 = 1 core).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=8000
	CPUMillis int64 `json:"cpuMillis"`

	// MemoryMi is the memory limit in mebibytes (e.g., 512 = 512 MiB).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=32
	// +kubebuilder:validation:Maximum=16384
	MemoryMi int64 `json:"memoryMi"`
}

// RuntimeStorage defines persistent storage configuration.
type RuntimeStorage struct {
	// SizeGi is the requested storage size in gibibytes.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	SizeGi int64 `json:"sizeGi"`

	// StorageClassName is the name of the StorageClass to use.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	StorageClassName string `json:"storageClassName"`
}

// RuntimeNetworkMode defines the ingress exposure mode.
// +kubebuilder:validation:Enum=path;subdomain
type RuntimeNetworkMode string

const (
	RuntimeNetworkModePath      RuntimeNetworkMode = "path"
	RuntimeNetworkModeSubdomain RuntimeNetworkMode = "subdomain"
)

// RuntimeNetwork defines network exposure configuration.
type RuntimeNetwork struct {
	// Mode controls how the runtime is exposed.
	// +kubebuilder:validation:Required
	Mode RuntimeNetworkMode `json:"mode"`

	// Host is the base hostname for the runtime.
	// For path mode: runtime.hermescloud.app
	// For subdomain mode: hermescloud.app
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Host string `json:"host"`

	// Path is the URL path for path mode (e.g., /rt-123).
	// Required when mode is "path", ignored when mode is "subdomain".
	// +optional
	Path string `json:"path,omitempty"`

	// Subdomain is the subdomain prefix for subdomain mode (e.g., rt-123).
	// Required when mode is "subdomain", ignored when mode is "path".
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Subdomain string `json:"subdomain,omitempty"`

	// IngressClassName specifies the IngressClass to use.
	// +optional
	IngressClassName string `json:"ingressClassName,omitempty"`
}

// RuntimeEnvVar defines an environment variable.
type RuntimeEnvVar struct {
	// Name is the environment variable name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Value is the environment variable value.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=0
	// +kubebuilder:validation:MaxLength=16384
	Value string `json:"value"`
}

// RuntimeSecretReference defines a reference to a Secret.
type RuntimeSecretReference struct {
	// Name is the Secret name in the runtime namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}

// RuntimePhase represents the current phase of the Runtime.
// +kubebuilder:validation:Enum=creating;running;failed;deleting;deleted
type RuntimePhase string

const (
	RuntimePhaseCreating RuntimePhase = "creating"
	RuntimePhaseRunning  RuntimePhase = "running"
	RuntimePhaseFailed   RuntimePhase = "failed"
	RuntimePhaseDeleting RuntimePhase = "deleting"
	RuntimePhaseDeleted  RuntimePhase = "deleted"
)

// RuntimeSpec defines the desired state of Runtime
type RuntimeSpec struct {
	// TenantID is the tenant identifier from the application database.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	TenantID string `json:"tenantId"`

	// RuntimeID is the runtime identifier from the application database.
	// It must be a DNS-1123 label because it is embedded in Kubernetes resource names.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=48
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	RuntimeID string `json:"runtimeId"`

	// Image is the fully qualified OCI image reference for the runtime workload.
	// Prefer immutable tags or digests.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Image string `json:"image"`

	// Plan controls runtime CPU and memory limits.
	// +kubebuilder:validation:Required
	Plan RuntimePlan `json:"plan"`

	// Storage controls the runtime PVC.
	// +kubebuilder:validation:Required
	Storage RuntimeStorage `json:"storage"`

	// Network controls ingress exposure.
	// +kubebuilder:validation:Required
	Network RuntimeNetwork `json:"network"`

	// Config is rendered into /opt/data/config.yaml.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Config k8sruntime.RawExtension `json:"config,omitempty"`

	// Soul is rendered into /opt/data/SOUL.md.
	// +optional
	// +kubebuilder:validation:MaxLength=65535
	Soul string `json:"soul,omitempty"`

	// Env defines non-secret environment variables to inject into the runtime container.
	// +listType=map
	// +listMapKey=name
	// +optional
	Env []RuntimeEnvVar `json:"env,omitempty"`

	// EnvFromSecrets lists Secrets that already exist in the runtime namespace and
	// should be exposed to the container with envFrom.
	// +listType=map
	// +listMapKey=name
	// +optional
	EnvFromSecrets []RuntimeSecretReference `json:"envFromSecrets,omitempty"`
}

// RuntimeStatus defines the observed state of Runtime.
type RuntimeStatus struct {
	// Phase represents the current high-level state of the Runtime.
	// +optional
	Phase RuntimePhase `json:"phase,omitempty"`

	// URL is the public endpoint for accessing the runtime.
	// +optional
	URL string `json:"url,omitempty"`

	// Namespace is the namespace where the runtime resources are deployed.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// DeploymentName is the name of the managed Deployment.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// ServiceName is the name of the managed Service.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// IngressName is the name of the managed Ingress.
	// +optional
	IngressName string `json:"ingressName,omitempty"`

	// ReadyReplicas is the number of ready replicas in the Deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// DesiredReplicas is the desired number of replicas in the Deployment.
	// +optional
	DesiredReplicas int32 `json:"desiredReplicas,omitempty"`

	// Message provides human-readable information about the current state.
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represent the current state of the Runtime resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastReconciledAt records when the controller last processed this resource.
	// +optional
	LastReconciledAt *metav1.Time `json:"lastReconciledAt,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:csv:customresourcedefinitions:displayName="Hermes Runtime"
// +operator-sdk:csv:customresourcedefinitions:description="Represents a single Hermes agent runtime and the desired configuration used to reconcile its Kubernetes workload and public endpoint."
// +operator-sdk:csv:customresourcedefinitions:resources={{Deployment,apps/v1},{ConfigMap,v1},{PersistentVolumeClaim,v1},{Secret,v1},{Service,v1},{Ingress,networking.k8s.io/v1}}

// Runtime is the Schema for the runtimes API
type Runtime struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Runtime
	// +required
	Spec RuntimeSpec `json:"spec"`

	// status defines the observed state of Runtime
	// +optional
	Status RuntimeStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RuntimeList contains a list of Runtime
type RuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Runtime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runtime{}, &RuntimeList{})
}
