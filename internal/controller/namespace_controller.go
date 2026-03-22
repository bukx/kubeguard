package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubeguardv1alpha1 "github.com/bukx/kubeguard/api/v1alpha1"
)

// NamespaceReconciler reconciles a ManagedNamespace object.
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=managednamespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeguard.bukx.dev,resources=managednamespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=limitranges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles ManagedNamespace reconciliation.
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ManagedNamespace resource
	managed := &kubeguardv1alpha1.ManagedNamespace{}
	if err := r.Get(ctx, req.NamespacedName, managed); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling ManagedNamespace",
		"name", managed.Name,
		"owner", managed.Spec.Owner,
		"environment", managed.Spec.Environment,
	)

	// Check if namespace has expired
	if managed.Spec.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, managed.Spec.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			logger.Info("Namespace has expired, cleaning up", "name", managed.Name)
			managed.Status.Phase = "Expired"

			// Delete the namespace
			ns := &corev1.Namespace{}
			if err := r.Get(ctx, client.ObjectKey{Name: managed.Name}, ns); err == nil {
				if err := r.Delete(ctx, ns); err != nil {
					logger.Error(err, "Failed to delete expired namespace")
					return ctrl.Result{}, err
				}
			}

			if err := r.Status().Update(ctx, managed); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	// Create or update namespace
	if err := r.reconcileNamespace(ctx, managed); err != nil {
		return ctrl.Result{}, err
	}

	// Create ResourceQuota
	if managed.Spec.ResourceQuota != nil {
		if err := r.reconcileResourceQuota(ctx, managed); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create LimitRange
	if managed.Spec.LimitRange != nil {
		if err := r.reconcileLimitRange(ctx, managed); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create NetworkPolicy
	if managed.Spec.NetworkPolicy != nil {
		if err := r.reconcileNetworkPolicy(ctx, managed); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create RBAC bindings
	if len(managed.Spec.RBAC) > 0 {
		if err := r.reconcileRBAC(ctx, managed); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status
	managed.Status.Phase = "Active"
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "Reconciled",
		Message:            "All namespace resources have been created",
	}
	setCondition(&managed.Status.Conditions, condition)

	if err := r.Status().Update(ctx, managed); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to check expiration
	return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
}

func (r *NamespaceReconciler) reconcileNamespace(ctx context.Context, managed *kubeguardv1alpha1.ManagedNamespace) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: managed.Name,
			Labels: map[string]string{
				"kubeguard.bukx.dev/managed":     "true",
				"kubeguard.bukx.dev/owner":       managed.Spec.Owner,
				"kubeguard.bukx.dev/environment": managed.Spec.Environment,
			},
		},
	}

	existing := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: managed.Name}, existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, ns)
		}
		return err
	}

	// Update labels on existing namespace
	existing.Labels = ns.Labels
	return r.Update(ctx, existing)
}

func (r *NamespaceReconciler) reconcileResourceQuota(ctx context.Context, managed *kubeguardv1alpha1.ManagedNamespace) error {
	hard := make(corev1.ResourceList)
	for k, v := range managed.Spec.ResourceQuota.Hard {
		hard[corev1.ResourceName(k)] = resource.MustParse(v)
	}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-quota", managed.Name),
			Namespace: managed.Name,
			Labels: map[string]string{
				"kubeguard.bukx.dev/managed": "true",
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hard,
		},
	}

	existing := &corev1.ResourceQuota{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(quota), existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, quota)
		}
		return err
	}
	existing.Spec = quota.Spec
	return r.Update(ctx, existing)
}

func (r *NamespaceReconciler) reconcileLimitRange(ctx context.Context, managed *kubeguardv1alpha1.ManagedNamespace) error {
	defaultLimit := make(corev1.ResourceList)
	defaultRequest := make(corev1.ResourceList)

	for k, v := range managed.Spec.LimitRange.DefaultLimit {
		defaultLimit[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	for k, v := range managed.Spec.LimitRange.DefaultRequest {
		defaultRequest[corev1.ResourceName(k)] = resource.MustParse(v)
	}

	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-limits", managed.Name),
			Namespace: managed.Name,
			Labels: map[string]string{
				"kubeguard.bukx.dev/managed": "true",
			},
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type:           corev1.LimitTypeContainer,
					Default:        defaultLimit,
					DefaultRequest: defaultRequest,
				},
			},
		},
	}

	existing := &corev1.LimitRange{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(lr), existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, lr)
		}
		return err
	}
	existing.Spec = lr.Spec
	return r.Update(ctx, existing)
}

func (r *NamespaceReconciler) reconcileNetworkPolicy(ctx context.Context, managed *kubeguardv1alpha1.ManagedNamespace) error {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-default", managed.Name),
			Namespace: managed.Name,
			Labels: map[string]string{
				"kubeguard.bukx.dev/managed": "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// Configure ingress rules
	if managed.Spec.NetworkPolicy.AllowIngress {
		ingressRule := networkingv1.NetworkPolicyIngressRule{}
		if len(managed.Spec.NetworkPolicy.AllowedNamespaces) > 0 {
			for _, ns := range managed.Spec.NetworkPolicy.AllowedNamespaces {
				ingressRule.From = append(ingressRule.From, networkingv1.NetworkPolicyPeer{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": ns,
						},
					},
				})
			}
		}
		policy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{ingressRule}
	}

	// Configure egress rules
	if managed.Spec.NetworkPolicy.AllowEgress {
		policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{{}}
	}

	existing := &networkingv1.NetworkPolicy{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(policy), existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, policy)
		}
		return err
	}
	existing.Spec = policy.Spec
	return r.Update(ctx, existing)
}

func (r *NamespaceReconciler) reconcileRBAC(ctx context.Context, managed *kubeguardv1alpha1.ManagedNamespace) error {
	for i, binding := range managed.Spec.RBAC {
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-rb-%d", managed.Name, i),
				Namespace: managed.Name,
				Labels: map[string]string{
					"kubeguard.bukx.dev/managed": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     binding.Role,
			},
		}

		for _, subject := range binding.Subjects {
			rb.Subjects = append(rb.Subjects, rbacv1.Subject{
				Kind:      subject.Kind,
				Name:      subject.Name,
				Namespace: subject.Namespace,
			})
		}

		existing := &rbacv1.RoleBinding{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(rb), existing); err != nil {
			if errors.IsNotFound(err) {
				if err := r.Create(ctx, rb); err != nil {
					return err
				}
				continue
			}
			return err
		}
		existing.RoleRef = rb.RoleRef
		existing.Subjects = rb.Subjects
		if err := r.Update(ctx, existing); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeguardv1alpha1.ManagedNamespace{}).
		Owns(&corev1.Namespace{}).
		Named("namespace").
		Complete(r)
}
