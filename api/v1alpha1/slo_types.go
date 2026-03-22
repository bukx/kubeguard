package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BurnRateThreshold defines a multi-window burn rate alert threshold.
type BurnRateThreshold struct {
	// Severity of the alert (critical, warning, info).
	Severity string `json:"severity"`
	// ShortWindow is the short evaluation window (e.g., "5m").
	ShortWindow string `json:"shortWindow"`
	// LongWindow is the long evaluation window (e.g., "1h").
	LongWindow string `json:"longWindow"`
	// Factor is the burn rate multiplier that triggers the alert.
	Factor string `json:"factor"`
}

// SLOAlerting configures alert generation for the SLO.
type SLOAlerting struct {
	// BurnRateThresholds defines multi-window, multi-burn-rate alert thresholds.
	BurnRateThresholds []BurnRateThreshold `json:"burnRateThresholds,omitempty"`
}

// SLOIndicator defines how the SLI is measured.
type SLOIndicator struct {
	// Type is the SLI type (availability, latency).
	// +kubebuilder:validation:Enum=availability;latency
	Type string `json:"type"`
	// TotalQuery is the PromQL query for total requests.
	TotalQuery string `json:"totalQuery"`
	// ErrorQuery is the PromQL query for error requests.
	ErrorQuery string `json:"errorQuery,omitempty"`
	// ThresholdQuery is the PromQL query for latency threshold (used when type=latency).
	ThresholdQuery string `json:"thresholdQuery,omitempty"`
}

// ServiceLevelObjectiveSpec defines the desired state of a ServiceLevelObjective.
type ServiceLevelObjectiveSpec struct {
	// Service is the name of the service this SLO applies to.
	Service string `json:"service"`
	// Description is a human-readable description of the SLO.
	Description string `json:"description,omitempty"`
	// Target is the SLO target percentage (e.g., "99.9").
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?$`
	Target string `json:"target"`
	// Window is the SLO evaluation window (e.g., "30d", "7d").
	Window string `json:"window"`
	// Indicator defines how the SLI is measured.
	Indicator SLOIndicator `json:"indicator"`
	// Alerting configures alert generation.
	Alerting *SLOAlerting `json:"alerting,omitempty"`
}

// ServiceLevelObjectiveStatus defines the observed state of a ServiceLevelObjective.
type ServiceLevelObjectiveStatus struct {
	// CurrentCompliance is the current SLO compliance percentage.
	CurrentCompliance string `json:"currentCompliance,omitempty"`
	// ErrorBudgetRemaining is the remaining error budget (0-100%).
	ErrorBudgetRemaining string `json:"errorBudgetRemaining,omitempty"`
	// LastCalculation is the timestamp of the last SLO calculation.
	LastCalculation *metav1.Time `json:"lastCalculation,omitempty"`
	// PrometheusRuleRef is the name of the generated PrometheusRule.
	PrometheusRuleRef string `json:"prometheusRuleRef,omitempty"`
	// Conditions represent the latest available observations of the SLO's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.service`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.target`
// +kubebuilder:printcolumn:name="Compliance",type=string,JSONPath=`.status.currentCompliance`
// +kubebuilder:printcolumn:name="Budget",type=string,JSONPath=`.status.errorBudgetRemaining`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ServiceLevelObjective is the Schema for the servicelevelobjectives API.
type ServiceLevelObjective struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLevelObjectiveSpec   `json:"spec,omitempty"`
	Status ServiceLevelObjectiveStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceLevelObjectiveList contains a list of ServiceLevelObjective.
type ServiceLevelObjectiveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLevelObjective `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceLevelObjective{}, &ServiceLevelObjectiveList{})
}
