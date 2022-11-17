package analyzer

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type Result struct {
	ResourceType string
	ResourceName string
	Healthy      bool
	Errors       []string
}

// Pod analyzes a pod object for potential errors and returns a Result
func Pod(pod *v1.Pod) Result {
	result := Result{
		ResourceType: "Pod",
		ResourceName: pod.Name,
		Healthy:      true,
	}
	// Check the conditions
	if pod.Status.Phase != v1.PodSucceeded {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				result.Healthy = false
				break
			}
		}
	}
	// Check init container statuses.
	analyzeContainers(pod.Status.InitContainerStatuses, &result)
	// Check container statuses. Note that if any init containers are not ready, the non-init containers will be stuck in a waiting state.
	analyzeContainers(pod.Status.ContainerStatuses, &result)
	return result
}

func analyzeContainers(containerStatuses []v1.ContainerStatus, result *Result) {
	for _, container := range containerStatuses {
		if !container.Ready {
			result.Healthy = false
			if container.State.Waiting != nil {
				result.Errors = append(result.Errors, formatContainerError(container.Name, "waiting", container.State.Waiting.Reason, container.State.Waiting.Message, 0, container.Ready))
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "Completed" {
				result.Errors = append(result.Errors, formatContainerError(container.Name, "terminated", container.State.Terminated.Reason, container.State.Terminated.Message, container.State.Terminated.ExitCode, container.Ready))
			} else if container.State.Running != nil && !container.Ready {
				result.Errors = append(result.Errors, formatContainerError(container.Name, "running", "", "", 0, container.Ready))
			}
		}
	}
}

func formatContainerError(containerName, state, reason, message string, exitCode int32, ready bool) string {
	sb := new(strings.Builder)
	if len(containerName) > 0 {
		sb.WriteString(fmt.Sprintf("container '%s'", containerName))
	}
	if !ready {
		sb.WriteString(" is not ready and")
		if len(state) > 0 {
			sb.WriteString(fmt.Sprintf(" is in a %s state", state))
		}
		if len(reason) > 0 {
			sb.WriteString(fmt.Sprintf(" due to reason '%s'", reason))
		}
		if len(message) > 0 {
			sb.WriteString(fmt.Sprintf(" with message '%s'", message))
		}
		if exitCode > 0 {
			sb.WriteString(fmt.Sprintf(" (exit code %d)", exitCode))
		}
	}
	return strings.TrimSpace(sb.String())

}
