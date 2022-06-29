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
//
// Usage:
//	c, err := k8s.CreateClient()
//	if err != nil {
//		panic(err)
//	}
//	pods, err := k8s.GetPods(c, "eg-api-gateway", "")
//	if err != nil {
//		panic(err)
//	}
//	for _, pod := range pods {
//		result := analyzer.Pod(pod)
//		log.Printf("%+v", result)
//	}
func Pod(pod v1.Pod) Result {
	result := Result{
		ResourceType: "Pod",
		ResourceName: pod.Name,
		Healthy:      true,
	}
	// Check the conditions (unless it's a job or a finite
	if pod.Status.Phase != v1.PodSucceeded {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				result.Healthy = false
				break
			}
		}
	}
	// Check the container states
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running != nil && (container.State.Terminated != nil && container.State.Terminated.Reason != "Completed") {
			result.Healthy = false
		}
		if container.State.Waiting != nil {
			result.Errors = append(result.Errors, formatContainerError(container.Name, "waiting", container.State.Waiting.Reason, container.State.Waiting.Message, 0))
		} else if container.State.Terminated != nil && container.State.Terminated.Reason != "Completed" {
			result.Errors = append(result.Errors, formatContainerError(container.Name, "terminated", container.State.Terminated.Reason, container.State.Terminated.Message, container.State.Terminated.ExitCode))
		}
	}
	return result
}

func formatContainerError(containerName, state, reason, message string, exitCode int32) string {
	sb := new(strings.Builder)
	if len(containerName) > 0 {
		sb.WriteString(fmt.Sprintf("container '%s'", containerName))
	}
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
	return strings.TrimSpace(sb.String())
}
