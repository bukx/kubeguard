package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceQuotaSpec defines resource quota for the namespace.
type ResourceQuotaSpec struct {
	// Hard is the set of desired hard limits for each named resource.
	Hard map[string]string `json:"hard,omitempty"`
}

// LimitRangeDefaults defines default limits and requests for containers.
type LimitRangeDefaults struct {
	// DefaultLimit is the default resource limit per container.
	DefaultLimit map[string]string `json:"defaultLimit,omitempty"`
	// DefaultRequest is the default resource request per container.
	DefaultRequest map[string]string `json:"defaultRequest,omitempty"`
}

// NetworkPolicySpec defines network isolation rules.
type NetworkPolicySpec struct {
	// AllowIngress allows inbound traffic to the namespace.
	AllowIngress bool `json:"allowIngress,omitempty"`
	// AllowEgress allows outbound traffic from the namespace.
	AllowEgress bool `json:"allowEgress,omitempty"`
	// AllowedNamespaces is a list of namespaces allowed to communicate.
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`
}

// RBACSubject defines a subject for RBAC binding.
type RBACSubject struct {
	// Kind is the type of subject (User, Group, ServiceAccount).
	Kind string `json:"kind"`
	// Name is the name of the subject.
	Name string `json:"name"`
	// Namespace is the namespace of the subject (for ServiceAccount).
	Namespace string `json:"namespace,omitempty"`
}

// RBACBinding defines a role binding for the namespace.
type RBACBinding struct {
	// Role is the ClusterRole to bind (admin, edit, view).
	Role string `json:"role"`
	// Subjects is the list of subjects to bind the role to.
	Subjects []RBACSubject `json:"subjects"`
}

// MonitoringSpec defines monitoring configuration for the namespace.
type MonitoringSpec struct {
	// EnablePrometheus creates a ServiceMonitor for the namespace.
	EnablePrometheus bool `json:"enablePrometheus,omitempty"`
	// AlertEmail is the email address for namespace-scoped alerts.
	AlertEmail string `json:"alertEmail,omitempty"`
}

// ManagedNamespaceSpec defines the desired state of ManagedNamespace.
type ManagedNamespaceSpec struct {
	// Owner is the team or individual that owns this namespace.
	Owner string `json:"owner"`
	// Environment is the environment label (production, staging, dev).
	// +kubebuilder:validation:Enum=production;staging;dev;sandbox
	Environment string `json:"environment"`
	// ExpiresAt is the optional TTL for the namespace (RFC3339 format).
	ExpiresAt string `json:"expiresAt,omitempty"`
	// ResourceQuota defines resource limits for the namespace.
	ResourceQuota *ResourceQuotaSpec `json:"resourceQuota,omitempty"`
	// LimitRange defines default container limits.
	LimitRange *LimitRangeDefaults `json:"limitRange,omitempty"`
	// NetworkPolicy defines network isolation rules.
	NetworkPolicy *NetworkPolicySpec `json:"networkPolicy,omitempty"`
	// RBAC defines role bindings for the namespace.
	RBAC []RBACBinding `json:"rbac,omitempty"`
	// Monitoring defines monitoring configuration.
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`
}

// ManagedNamespaceStatus defines the observed state of ManagedNamespace.
type ManagedNamespaceStatus struct {
	// Phase is the current phase of the namespace (Active, Expiring, Deleted).
	Phase string `json:"phase,omitempty"`
	// ResourcesCreated lists all resources created for this namespace.
	ResourcesCreated []string `json:"resourcesCreated,omitempty"`
	// ExpiresIn is the human-readable time until expiration.
	ExpiresIn string `json:"expiresIn,omitempty"`
	// Conditions represent the latest available observations.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
// +kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Expires",type=string,JSONPath=`.status.expiresIn`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ManagedNamespace is the Schema for the managednamespaces API.
type ManagedNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedNamespaceSpec   `json:"spec,omitempty"`
	Status ManagedNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedNamespaceList contains a list of ManagedNamespace.
type ManagedNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedNamespace{}, &ManagedNamespaceList{})
}
