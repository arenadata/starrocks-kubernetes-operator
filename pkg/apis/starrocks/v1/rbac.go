/*
Copyright 2021-present, StarRocks Inc.

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

// MANAGED role: the permissions the operator needs in every WATCHED namespace to create and
// manage the resources that back a StarRocks cluster/warehouse. These markers are scanned by
// `controller-gen rbac ... paths=./pkg/apis/...` and rendered into the operator chart
// managed_role.yaml with a namespace of {{ . }} (templated per watched namespace).
//
// This file intentionally contains only +kubebuilder:rbac markers and no Go code.

// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// TODO: AD use secrets instead configmaps
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=pods,verbs=get;list;watch
// TODO: AD apps/v1 expose pod endpoints by selectorLabels
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ . }}",groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:namespace="{{ . }}",groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace="{{ . }}",groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace="{{ . }}",groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace="{{ . }}",groups=starrocks.com,resources=starrocksclusters,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ . }}",groups=starrocks.com,resources=starrocksclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace="{{ . }}",groups=starrocks.com,resources=starrockswarehouses,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ . }}",groups=starrocks.com,resources=starrockswarehouses/status,verbs=get;update;patch
