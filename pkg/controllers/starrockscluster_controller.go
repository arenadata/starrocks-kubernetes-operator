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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/be"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/cn"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/fe"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/feproxy"
)

// StarRocksClusterReconciler reconciles a StarRocksCluster object
type StarRocksClusterReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scs      []subcontrollers.ClusterSubController
}

// MANAGER role: the operator only needs the following permissions in its OWN namespace.
// These markers are scanned by `controller-gen rbac ... paths=./pkg/controllers` and rendered
// into the operator chart manager_role.yaml with a namespace of {{ .Release.Namespace }}.

// +kubebuilder:rbac:namespace="{{ .Release.Namespace }}",groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:namespace="{{ .Release.Namespace }}",groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace="{{ .Release.Namespace }}",groups=starrocks.com,resources=starrocksclusters,verbs=get;list;watch
// +kubebuilder:rbac:namespace="{{ .Release.Namespace }}",groups=starrocks.com,resources=starrocksclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace="{{ .Release.Namespace }}",groups=starrocks.com,resources=starrocksclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *StarRocksClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.Log.WithName("StarRocksClusterReconciler").WithValues("name", req.Name, "namespace", req.Namespace)
	ctx = logr.NewContext(ctx, logger)
	logger.Info("begin to reconcile StarRocksCluster")

	logger.Info("get StarRocksCluster CR from kubernetes")

	esrc := new(srapi.StarRocksCluster)
	err := r.Get(ctx, req.NamespacedName, esrc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "get StarRocksCluster object failed")
		return requeueIfError(err)
	}

	// reconcile src deleted
	if !esrc.DeletionTimestamp.IsZero() {
		logger.Info("deletion timestamp is not zero, clear StarRocksCluster related resources")
		return ctrl.Result{}, nil
	}

	// subControllers reconcile for create or update component.
	for _, rc := range r.Scs {
		kvs := []interface{}{"subController", rc.GetControllerName()}
		logger.Info("sub controller sync spec", kvs...)
		if err = rc.SyncCluster(ctx, esrc); err != nil {
			logger.Error(err, "sub controller reconciles spec failed", kvs...)
			handleSyncClusterError(esrc, rc, err)
			if updateError := r.UpdateStarRocksClusterStatus(ctx, esrc); updateError != nil {
				logger.Error(updateError, "failed to update StarRocksCluster Status")
			}
			return requeueIfError(err)
		}
	}

	for _, rc := range r.Scs {
		kvs := []interface{}{"subController", rc.GetControllerName()}
		logger.Info("sub controller update status", kvs...)
		if err = rc.UpdateClusterStatus(ctx, esrc); err != nil {
			logger.Error(err, "sub controller update status failed", kvs...)
			handleSyncClusterError(esrc, rc, err)
			if updateError := r.UpdateStarRocksClusterStatus(ctx, esrc); updateError != nil {
				logger.Error(updateError, "failed to update StarRocksCluster Status")
			}
			return requeueIfError(err)
		}
	}

	logger.Info("update StarRocksCluster level status")
	r.reconcileStatus(ctx, esrc)
	err = r.UpdateStarRocksClusterStatus(ctx, esrc)
	if err != nil {
		logger.Error(err, "update StarRocksCluster status failed")
		return ctrl.Result{}, err
	}
	logger.Info("reconcile StarRocksCluster success")
	return ctrl.Result{}, nil
}

// UpdateStarRocksClusterStatus update the status of src.
func (r *StarRocksClusterReconciler) UpdateStarRocksClusterStatus(ctx context.Context, src *srapi.StarRocksCluster) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		esrc := new(srapi.StarRocksCluster)
		if err := r.Get(ctx, types.NamespacedName{Namespace: src.Namespace, Name: src.Name}, esrc); err != nil {
			return err
		}

		esrc.Status = src.Status
		return r.Status().Update(ctx, esrc)
	})
}

func (r *StarRocksClusterReconciler) reconcileStatus(_ context.Context, src *srapi.StarRocksCluster) {
	src.Status.Phase = srapi.ClusterRunning
	src.Status.Reason = ""
	var phase srapi.Phase
	if src.Status.StarRocksFeStatus != nil {
		phase = GetPhaseFromComponent(&src.Status.StarRocksFeStatus.StarRocksComponentStatus)
		if phase != "" {
			src.Status.Phase = phase
			return
		}
	}
	if src.Status.StarRocksBeStatus != nil {
		phase = GetPhaseFromComponent(&src.Status.StarRocksBeStatus.StarRocksComponentStatus)
		if phase != "" {
			src.Status.Phase = phase
			return
		}
	}
	if src.Status.StarRocksCnStatus != nil {
		phase = GetPhaseFromComponent(&src.Status.StarRocksCnStatus.StarRocksComponentStatus)
		if phase != "" {
			src.Status.Phase = phase
			return
		}
	}
}

// handleSyncClusterError handle errors from sub-controller, and log it in StarRocksCluster Status
func handleSyncClusterError(src *srapi.StarRocksCluster, subController subcontrollers.ClusterSubController, err error) {
	reason := err.Error()
	switch subController.(type) {
	case *fe.FeController:
		reason = fmt.Sprintf("error from FE controller: %v", reason)
	case *be.BeController:
		reason = fmt.Sprintf("error from BE controller: %v", reason)
	case *cn.CnController:
		reason = fmt.Sprintf("error from CN controller: %v", reason)
	case *feproxy.FeProxyController:
		reason = fmt.Sprintf("error from fe-proxy controller: %v", reason)
	}

	src.Status.Phase = srapi.ClusterFailed
	src.Status.Reason = reason
}

func requeueIfError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// GetPhaseFromComponent return the Phase of Cluster or Warehouse based on the component status.
// It returns empty string if not sure the phase.
func GetPhaseFromComponent(componentStatus *srapi.StarRocksComponentStatus) srapi.Phase {
	if componentStatus == nil {
		return ""
	}
	if componentStatus.Phase == srapi.ComponentReconciling {
		return srapi.ClusterReconciling
	}
	if componentStatus.Phase == srapi.ComponentFailed {
		return srapi.ClusterFailed
	}
	return ""
}
