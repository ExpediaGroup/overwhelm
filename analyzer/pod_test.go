package analyzer

import (
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod(t *testing.T) {
	scenarios := []struct {
		name string
		pod  v1.Pod
		want Result
	}{
		{
			name: "healthy",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: v1.PodInitialized, Status: v1.ConditionTrue},
						{Type: v1.PodReady, Status: v1.ConditionTrue},
						{Type: v1.ContainersReady, Status: v1.ConditionTrue},
						{Type: v1.PodScheduled, Status: v1.ConditionTrue},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "application",
							Ready: true,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{
									StartedAt: metav1.Time{Time: time.Now().Add(-time.Minute)},
								},
							},
						},
					},
				},
			},
			want: Result{
				ResourceType: "Pod",
				ResourceName: "pod-name",
				Healthy:      true,
				Errors:       nil,
			},
		},
		{
			name: "unhealthy",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: v1.PodReady, Status: v1.ConditionFalse},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "application",
							Ready: false,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "PodInitializing",
								},
							},
						},
					},
				},
			},
			want: Result{
				ResourceType: "Pod",
				ResourceName: "pod-name",
				Healthy:      false,
				Errors:       []string{"container 'application' is not ready and is in a waiting state due to reason 'PodInitializing'"},
			},
		},
		{
			name: "init-container-preventing-other-containers-from-starting",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: v1.PodInitialized, Status: v1.ConditionFalse, Reason: "ContainersNotInitialized", Message: "containers with incomplete status: [vault-agent-init istio-init]"},
						{Type: v1.PodReady, Status: v1.ConditionFalse, Reason: "ContainersNotReady", Message: "containers with unready status: [istio-proxy application vault-agent]"},
						{Type: v1.ContainersReady, Status: v1.ConditionFalse, Reason: "ContainersNotReady", Message: "containers with unready status: [istio-proxy application vault-agent]"},
						{Type: v1.PodScheduled, Status: v1.ConditionTrue},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "application",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{Reason: "PodInitializing"},
							},
						},
						{
							Name:         "istio-proxy",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{Reason: "PodInitializing"},
							},
						},
						{
							Name:         "vault-agent",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{Reason: "PodInitializing"},
							},
						},
					},
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "vault-agent-init",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{StartedAt: metav1.Time{Time: time.Now().Add(-time.Minute)}},
							},
						},
						{
							Name:         "istio-init",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{Reason: "PodInitializing"},
							},
						},
					},
				},
			},
			want: Result{
				ResourceType: "Pod",
				ResourceName: "pod-name",
				Healthy:      false,
				Errors: []string{
					"container 'vault-agent-init' is not ready and is in a running state",
					"container 'istio-init' is not ready and is in a waiting state due to reason 'PodInitializing'",
					"container 'application' is not ready and is in a waiting state due to reason 'PodInitializing'",
					"container 'istio-proxy' is not ready and is in a waiting state due to reason 'PodInitializing'",
					"container 'vault-agent' is not ready and is in a waiting state due to reason 'PodInitializing'",
				},
			},
		},
		{
			name: "application-container-crashing",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: v1.PodInitialized, Status: v1.ConditionTrue},
						{Type: v1.PodReady, Status: v1.ConditionFalse, Reason: "ContainersNotReady", Message: "containers with unready status: [application]"},
						{Type: v1.ContainersReady, Status: v1.ConditionFalse, Reason: "ContainersNotReady", Message: "containers with unready status: [application]"},
						{Type: v1.PodScheduled, Status: v1.ConditionTrue},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "istio-proxy",
							Ready:        true,
							RestartCount: 0,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{StartedAt: metav1.Time{Time: time.Now().Add(-time.Minute)}},
							},
						},
						{
							Name:         "application",
							Ready:        false,
							RestartCount: 0,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: `Back-off pulling image "secret/secret:secret"`},
							},
						},
					},
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "istio-init",
							Ready:        true,
							RestartCount: 0,
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{Reason: "Completed"},
							},
						},
					},
					Phase: "Pending",
				},
			},
			want: Result{
				ResourceType: "Pod",
				ResourceName: "pod-name",
				Healthy:      false,
				Errors: []string{
					`container 'application' is not ready and is in a waiting state due to reason 'ImagePullBackOff' with message 'Back-off pulling image "secret/secret:secret"'`,
				},
			},
		},
	}
	// Run the tests
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			got := Pod(&scenario.pod)
			if got.ResourceName != scenario.want.ResourceName {
				t.Errorf("expected result with ResourceName=%v, got ResourceName=%v", scenario.want.ResourceName, got.ResourceName)
			}
			if got.ResourceType != scenario.want.ResourceType {
				t.Errorf("expected result with ResourceType=%v, got ResourceType=%v", scenario.want.ResourceType, got.ResourceType)
			}
			if got.Healthy != scenario.want.Healthy {
				t.Errorf("expected result with Healthy=%v, got Healthy=%v", scenario.want.Healthy, got.Healthy)
			}
			if strings.Join(got.Errors, ";") != strings.Join(scenario.want.Errors, ";") {
				t.Errorf("expected result with Errors=%v, got Errors=%v", strings.Join(scenario.want.Errors, "\n"), strings.Join(got.Errors, "\n"))
			}
		})
	}
}
