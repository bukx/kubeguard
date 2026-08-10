package webhook

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAllContainersIncludesEphemeralContainers(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "init"},
			},
			Containers: []corev1.Container{
				{Name: "app"},
			},
			EphemeralContainers: []corev1.EphemeralContainer{
				{
					EphemeralContainerCommon: corev1.EphemeralContainerCommon{
						Name:  "debugger",
						Image: "busybox:1.36",
					},
				},
			},
		},
	}

	containers := allContainers(pod)
	if len(containers) != 3 {
		t.Fatalf("expected 3 containers, got %d", len(containers))
	}

	names := []string{containers[0].Name, containers[1].Name, containers[2].Name}
	expected := []string{"init", "app", "debugger"}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("expected container %d to be %q, got %q", i, expected[i], names[i])
		}
	}
}

func TestIsLatestTag(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{
			name:     "implicit latest when tag missing",
			image:    "nginx",
			expected: true,
		},
		{
			name:     "explicit latest tag",
			image:    "nginx:latest",
			expected: true,
		},
		{
			name:     "versioned tag",
			image:    "ghcr.io/acme/api:v1.2.3",
			expected: false,
		},
		{
			name:     "registry with port and versioned tag",
			image:    "registry.example.com:5000/team/app:v1",
			expected: false,
		},
		{
			name:     "registry with port and no tag",
			image:    "registry.example.com:5000/team/app",
			expected: true,
		},
		{
			name:     "digest pinned image",
			image:    "registry.example.com:5000/team/app@sha256:0123456789abcdef",
			expected: false,
		},
		{
			name:     "tag plus digest remains immutable",
			image:    "registry.example.com:5000/team/app:latest@sha256:0123456789abcdef",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLatestTag(tt.image); got != tt.expected {
				t.Fatalf("expected %v for %q, got %v", tt.expected, tt.image, got)
			}
		})
	}
}
