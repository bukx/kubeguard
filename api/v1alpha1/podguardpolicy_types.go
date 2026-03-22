package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodGuardRules defines the enforcement rules for pods.
type PodGuardRules struct {
	// RequireResourceLimits rejects pods without CPU/memory limits.
	RequireResourceLimits bool `json:"requireResourceLimits,omitempty"`
	// RequireProbes rejects pods without liveness/readiness probes.
	RequireProbes bool `json:"requireProbes,omitempty"`
	// DisallowRoot rejects pods running as root.
	DisallowRoot bool `json:"disallowRoot,omitempty"`
	// DisallowPrivileged rejects privileged containers.
	DisallowPrivileged bool `json:"disallowPrivileged,omitempty"`
	// DisallowLatestTag warns or rejects pods using :latest image tag.
	DisallowLatestTag bool `json:"disallowLatestTag,omitempty"`
	// MaxCPULimit is the maximum CPU limit allowed per container.
	MaxCPULimit string `json:"maxCPULimit,omitempty"`
	// MaxMemoryLimit is the maximum memory limit allowed per container.
	MaxMemoryLimit string `json:"maxMemoryLimit,omitempty"`
}

// PodGuardPolicySpec defines the desired state of PodGuardPolicy.
type PodGuardPolicySpec struct {
	// Enforcement level: strict (reject), permissive (warn), audit (log only).
	// +kubebuilder:validation:Enum=strict;permissive;audit
	// +kubebuilder:default=strict
	Enforcement string `json:"enforcement"`
	// Rules defines which checks to enforce.
	Rules PodGuardRules `json:"rules"`
	// ExemptImages is a list of image patterns exempt from checks.
	ExemptImages []string `json:"exemptImages,omitempty"`
	// ExemptServiceAccounts is a list of service accounts exempt from checks.
	ExemptServiceAccounts []string `json:"exemptServiceAccounts,omitempty"`
}

// PodGuardPolicyStatus defines the observed state of PodGuardPolicy.
type PodGuardPolicyStatus struct {
	// PodsEvaluated is the total number of pods evaluated by this policy.
	PodsEvaluated int64 `json:"podsEvaluated,omitempty"`
	// PodsRejected is the total number of pods rejected by this policy.
	PodsRejected int64 `json:"podsRejected,omitempty"`
	// PodsMutated is the total number of pods mutated by this policy.
	PodsMutated int64 `json:"podsMutated,omitempty"`
	// LastEvaluated is the timestamp of the last pod evaluation.
	LastEvaluated *metav1.Time `json:"lastEvaluated,omitempty"`
	// Conditions represent the latest available observations.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Enforcement",type=string,JSONPath=`.spec.enforcement`
// +kubebuilder:printcolumn:name="Evaluated",type=integer,JSONPath=`.status.podsEvaluated`
// +kubebuilder:printcolumn:name="Rejected",type=integer,JSONPath=`.status.podsRejected`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PodGuardPolicy is the Schema for the podguardpolicies API.
type PodGuardPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodGuardPolicySpec   `json:"spec,omitempty"`
	Status PodGuardPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodGuardPolicyList contains a list of PodGuardPolicy.
type PodGuardPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodGuardPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodGuardPolicy{}, &PodGuardPolicyList{})
}
