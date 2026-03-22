package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubeguardv1alpha1 "github.com/bukx/kubeguard/api/v1alpha1"
)

// PodGuardReconciler reconciles a PodGuardPolicy object.
type PodGuardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=podguardpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=podguardpolicies/status,verbs=get;update;patch

// Reconcile handles PodGuardPolicy reconciliation.
// The actual pod validation is handled by the admission webhook.
// This controller manages the policy lifecycle and status updates.
func (r *PodGuardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	policy := &kubeguardv1alpha1.PodGuardPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling PodGuardPolicy",
		"name", policy.Name,
		"namespace", policy.Namespace,
		"enforcement", policy.Spec.Enforcement,
	)

	// Update status condition
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               "Active",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "PolicyActive",
		Message:            "PodGuardPolicy is active and enforcing rules",
	}
	setCondition(&policy.Status.Conditions, condition)

	if err := r.Status().Update(ctx, policy); err != nil {
		logger.Error(err, "Failed to update PodGuardPolicy status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodGuardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeguardv1alpha1.PodGuardPolicy{}).
		Named("podguard").
		Complete(r)
}
