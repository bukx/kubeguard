package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeguardv1alpha1 "github.com/bukx/kubeguard/api/v1alpha1"
)

// PodValidator validates and mutates pods based on PodGuardPolicy.
type PodValidator struct {
	Client  client.Client
	Decoder admission.Decoder
}

// Handle processes admission requests for pods.
func (v *PodValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := v.Decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	logger.Info("Validating pod",
		"name", pod.Name,
		"namespace", req.Namespace,
	)

	// Fetch PodGuardPolicy for the namespace
	policyList := &kubeguardv1alpha1.PodGuardPolicyList{}
	if err := v.Client.List(ctx, policyList, client.InNamespace(req.Namespace)); err != nil {
		logger.Error(err, "Failed to list PodGuardPolicies")
		return admission.Allowed("no policy found, allowing")
	}

	if len(policyList.Items) == 0 {
		return admission.Allowed("no PodGuardPolicy in namespace")
	}

	// Use the first policy (could be extended to merge multiple)
	policy := policyList.Items[0]

	// Check if service account is exempt
	for _, exempt := range policy.Spec.ExemptServiceAccounts {
		if pod.Spec.ServiceAccountName == exempt {
			return admission.Allowed("service account is exempt")
		}
	}

	var violations []string
	var warnings []string

	for _, container := range allContainers(pod) {
		prefix := fmt.Sprintf("container %q", container.Name)

		// Check resource limits
		if policy.Spec.Rules.RequireResourceLimits {
			if container.Resources.Limits.Cpu().IsZero() || container.Resources.Limits.Memory().IsZero() {
				violations = append(violations, fmt.Sprintf("%s: missing resource limits (CPU and memory limits are required)", prefix))
			}
		}

		// Check max CPU limit
		if policy.Spec.Rules.MaxCPULimit != "" && !container.Resources.Limits.Cpu().IsZero() {
			maxCPU := resource.MustParse(policy.Spec.Rules.MaxCPULimit)
			if container.Resources.Limits.Cpu().Cmp(maxCPU) > 0 {
				violations = append(violations, fmt.Sprintf("%s: CPU limit %s exceeds maximum %s", prefix, container.Resources.Limits.Cpu().String(), policy.Spec.Rules.MaxCPULimit))
			}
		}

		// Check max memory limit
		if policy.Spec.Rules.MaxMemoryLimit != "" && !container.Resources.Limits.Memory().IsZero() {
			maxMem := resource.MustParse(policy.Spec.Rules.MaxMemoryLimit)
			if container.Resources.Limits.Memory().Cmp(maxMem) > 0 {
				violations = append(violations, fmt.Sprintf("%s: memory limit %s exceeds maximum %s", prefix, container.Resources.Limits.Memory().String(), policy.Spec.Rules.MaxMemoryLimit))
			}
		}

		// Check probes
		if policy.Spec.Rules.RequireProbes {
			if container.LivenessProbe == nil {
				warnings = append(warnings, fmt.Sprintf("%s: missing liveness probe", prefix))
			}
			if container.ReadinessProbe == nil {
				warnings = append(warnings, fmt.Sprintf("%s: missing readiness probe", prefix))
			}
		}

		// Check running as root
		if policy.Spec.Rules.DisallowRoot {
			if container.SecurityContext == nil || container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
				if pod.Spec.SecurityContext == nil || pod.Spec.SecurityContext.RunAsNonRoot == nil || !*pod.Spec.SecurityContext.RunAsNonRoot {
					violations = append(violations, fmt.Sprintf("%s: must set runAsNonRoot=true", prefix))
				}
			}
		}

		// Check privileged
		if policy.Spec.Rules.DisallowPrivileged {
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				violations = append(violations, fmt.Sprintf("%s: privileged containers are not allowed", prefix))
			}
		}

		// Check latest tag
		if policy.Spec.Rules.DisallowLatestTag {
			if isLatestTag(container.Image) && !isExemptImage(container.Image, policy.Spec.ExemptImages) {
				warnings = append(warnings, fmt.Sprintf("%s: using :latest tag is discouraged, pin to a specific version", prefix))
			}
		}
	}

	// Apply enforcement policy
	switch policy.Spec.Enforcement {
	case "strict":
		if len(violations) > 0 {
			return admission.Denied(fmt.Sprintf("PodGuardPolicy violations:\n%s", strings.Join(violations, "\n")))
		}
		if len(warnings) > 0 {
			return admission.Allowed("").WithWarnings(warnings...)
		}
	case "permissive":
		allMessages := append(violations, warnings...)
		if len(allMessages) > 0 {
			return admission.Allowed("").WithWarnings(allMessages...)
		}
	case "audit":
		// Log only, always allow
		if len(violations) > 0 || len(warnings) > 0 {
			logger.Info("Audit mode - violations detected",
				"violations", violations,
				"warnings", warnings,
				"pod", pod.Name,
			)
		}
	}

	return admission.Allowed("")
}

// allContainers returns all containers (init + regular + ephemeral).
func allContainers(pod *corev1.Pod) []corev1.Container {
	var containers []corev1.Container
	containers = append(containers, pod.Spec.InitContainers...)
	containers = append(containers, pod.Spec.Containers...)
	return containers
}

// isLatestTag checks if an image uses the :latest tag or no tag.
func isLatestTag(image string) bool {
	if !strings.Contains(image, ":") {
		return true // No tag defaults to latest
	}
	parts := strings.Split(image, ":")
	return parts[len(parts)-1] == "latest"
}

// isExemptImage checks if an image matches any exempt pattern.
func isExemptImage(image string, exemptImages []string) bool {
	for _, exempt := range exemptImages {
		if strings.Contains(image, exempt) {
			return true
		}
	}
	return false
}
