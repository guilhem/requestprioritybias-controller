/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var podlog = logf.Log.WithName("pod-resource")

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithDefaulter(&PodPriorityRequestBias{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups=core,reinvocationPolicy=IfNeeded,resources=pods,verbs=create;update,versions=v1,name=mpod-prority-request-bias.barpilot.io,admissionReviewVersions=v1

type PodPriorityRequestBias struct {
}

func (p *PodPriorityRequestBias) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}

	podlog.Info("default", "name", pod.Name)

	if pod.Name == "" {
		podlog.Info("ignore pod without name")
		return nil
	}

	if pod.Spec.Priority == nil {
		// admission webhook will set the default value for the priority field
		// hide ourselves from the defaulting chain and wait for reinvocation
		podlog.Info("ignore pod without priority", "name", pod.Name)
		return nil
	}

	// Sum the CPU requests of all containers
	var cpuRequest int64
	for _, container := range pod.Spec.Containers {
		cpuRequest += container.Resources.Requests.Cpu().MilliValue()
	}

	podlog.Info("cpu request", "name", pod.Name, "cpuRequest", cpuRequest)

	// Add the CPU request to the pod priority
	priority := *pod.Spec.Priority + int32(cpuRequest)

	podlog.Info("set priority", "name", pod.Name, "priority", priority)

	*pod.Spec.Priority = priority

	return nil
}