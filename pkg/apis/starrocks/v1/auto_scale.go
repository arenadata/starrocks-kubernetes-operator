/*
Copyright 2022 StarRocks.

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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AutoScalingPolicy defines the auto scale
type AutoScalingPolicy struct {
	// the policy of autoscaling. operator use autoscaling v2.
	HPAPolicy *HPAPolicy `json:"hpaPolicy,omitempty"`

	// version represents the autoscaler version for cn service. valid values are v1 and v2. v2beta2 is
	// deprecated (removed from the kubernetes API server in 1.26) and, if set, is treated as v2.
	// +optional
	Version AutoScalerVersion `json:"version,omitempty"`

	// MinReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down. It defaults to 1 pod.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper limit for the number of pods that can be set by the autoscaler;
	// cannot be smaller than MinReplicas.
	MaxReplicas int32 `json:"maxReplicas"`
}

type HPAPolicy struct {
	// +optional
	// Metrics specifies how to scale based on a single metric
	// the struct copy from k8s.io/api/autoscaling/v2/types.go. the redundancy code will hide the restriction about
	// HorizontalPodAutoscaler version and kubernetes releases matching issue.
	// the splice will have unsafe.Pointer convert, so be careful to edit the struct fields.
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`

	// +optional
	// HorizontalPodAutoscalerBehavior configures the scaling behavior of the target.
	// the struct copy from k8s.io/api/autoscaling/v2/types.go. the redundancy code will hide the restriction about
	// HorizontalPodAutoscaler version and kubernetes releases matching issue.
	// the
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

type AutoScalerVersion string

const (
	// AutoScalerV1 the cn service use v1 autoscaler. Reference to https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
	AutoScalerV1 AutoScalerVersion = "v1"

	// AutoScalerV2 the cn service use v2. Reference to  https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
	AutoScalerV2 AutoScalerVersion = "v2"
)

// Complete completes the default value of AutoScalerVersion.
// All supported kubernetes clusters now serve autoscaling/v2, so when the version is empty we always
// default to AutoScalerV2. The major/minor parameters are kept for API compatibility but are no longer
// used to pick a version.
func (version AutoScalerVersion) Complete(_, _ string) AutoScalerVersion {
	if version != "" {
		return version
	}
	return AutoScalerV2
}

// CreateEmptyHPA create an empty HPA object based on the version information
func (version AutoScalerVersion) CreateEmptyHPA(major, minor string) client.Object {
	filledVersion := version.Complete(major, minor)

	var object client.Object
	switch filledVersion {
	case AutoScalerV1:
		object = &autoscalingv1.HorizontalPodAutoscaler{}
	default:
		// AutoScalerV2, AutoScalerV2Beta2 and any unrecognized version are served as autoscaling/v2.
		// v2beta2 was removed from the kubernetes API server in 1.26. This is consistent with BuildHPA,
		// which also builds a v2 HorizontalPodAutoscaler for any non-v1 version.
		object = &autoscalingv2.HorizontalPodAutoscaler{}
	}
	return object
}
