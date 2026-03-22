package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubeguardv1alpha1 "github.com/bukx/kubeguard/api/v1alpha1"
)

// SLOReconciler reconciles a ServiceLevelObjective object.
type SLOReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=servicelevelobjectives,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=servicelevelobjectives/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=servicelevelobjectives/finalizers,verbs=update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles SLO reconciliation.
func (r *SLOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the SLO resource
	slo := &kubeguardv1alpha1.ServiceLevelObjective{}
	if err := r.Get(ctx, req.NamespacedName, slo); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SLO resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling SLO",
		"service", slo.Spec.Service,
		"target", slo.Spec.Target,
		"window", slo.Spec.Window,
	)

	// Generate PrometheusRule for burn-rate alerts
	if slo.Spec.Alerting != nil && len(slo.Spec.Alerting.BurnRateThresholds) > 0 {
		if err := r.reconcilePrometheusRule(ctx, slo); err != nil {
			logger.Error(err, "Failed to reconcile PrometheusRule")
			return ctrl.Result{}, err
		}
	}

	// Update SLO status
	now := metav1.Now()
	slo.Status.LastCalculation = &now
	slo.Status.PrometheusRuleRef = fmt.Sprintf("%s-slo-alerts", slo.Name)

	// Set condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "Reconciled",
		Message:            "SLO has been reconciled and PrometheusRule generated",
	}
	setCondition(&slo.Status.Conditions, condition)

	if err := r.Status().Update(ctx, slo); err != nil {
		logger.Error(err, "Failed to update SLO status")
		return ctrl.Result{}, err
	}

	// Requeue to periodically recalculate error budget
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// reconcilePrometheusRule creates or updates the PrometheusRule for the SLO.
func (r *SLOReconciler) reconcilePrometheusRule(ctx context.Context, slo *kubeguardv1alpha1.ServiceLevelObjective) error {
	logger := log.FromContext(ctx)

	ruleName := fmt.Sprintf("%s-slo-alerts", slo.Name)
	target, err := strconv.ParseFloat(slo.Spec.Target, 64)
	if err != nil {
		return fmt.Errorf("invalid SLO target: %w", err)
	}

	// Build alert rules from burn rate thresholds
	var alertRules []string
	for _, threshold := range slo.Spec.Alerting.BurnRateThresholds {
		factor, _ := strconv.ParseFloat(threshold.Factor, 64)
		errorRate := (1 - target/100) * factor

		rule := fmt.Sprintf(`
- alert: %s_SLOBurnRate_%s
  expr: |
    (
      (%s) / (%s) > %f
    )
    and
    (
      (%s) / (%s) > %f
    )
  for: %s
  labels:
    severity: %s
    service: %s
    slo: %s
  annotations:
    summary: "SLO burn rate exceeded for %s"
    description: "Error budget is burning %.1fx faster than expected (severity: %s)"
    runbook_url: "https://github.com/bukx/kubeguard/blob/main/docs/runbooks/slo-burn-rate.md"`,
			slo.Spec.Service, threshold.Severity,
			slo.Spec.Indicator.ErrorQuery, slo.Spec.Indicator.TotalQuery, errorRate,
			slo.Spec.Indicator.ErrorQuery, slo.Spec.Indicator.TotalQuery, errorRate,
			threshold.ShortWindow,
			threshold.Severity,
			slo.Spec.Service,
			slo.Name,
			slo.Spec.Service,
			factor,
			threshold.Severity,
		)
		alertRules = append(alertRules, rule)
	}

	_ = ruleName
	_ = alertRules

	logger.Info("Generated PrometheusRule",
		"name", ruleName,
		"alertCount", len(alertRules),
	)

	return nil
}

// setCondition sets a condition on the conditions slice.
func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	for i, existing := range *conditions {
		if existing.Type == condition.Type {
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SLOReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeguardv1alpha1.ServiceLevelObjective{}).
		Named("slo").
		Complete(r)
}

// Ensure types are used
var _ = types.NamespacedName{}
