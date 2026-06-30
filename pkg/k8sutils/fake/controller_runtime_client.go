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

package fake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
)

// NewFakeClient creates a new fake Kubernetes client.
func NewFakeClient(scheme *runtime.Scheme, initObjs ...runtime.Object) client.Client {
	builder := fake.NewClientBuilder().
		WithRuntimeObjects(initObjs...).
		WithScheme(scheme)

	// The upgraded fake client no longer registers status subresources by default, so we declare
	// them explicitly to keep /status updates working in tests. Only register the StarRocks custom
	// resources when the provided scheme actually knows about them; some tests use a client-go-only
	// scheme, and WithStatusSubresource panics on types it cannot resolve to a GVK.
	var statusSubresources []client.Object
	for _, obj := range []client.Object{&srapi.StarRocksCluster{}, &srapi.StarRocksWarehouse{}} {
		if gvks, _, err := scheme.ObjectKinds(obj); err == nil && len(gvks) > 0 {
			statusSubresources = append(statusSubresources, obj)
		}
	}
	if len(statusSubresources) > 0 {
		builder = builder.WithStatusSubresource(statusSubresources...)
	}

	return builder.Build()
}
