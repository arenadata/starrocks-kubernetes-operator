package controllers

import (
	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/predicates"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/be"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/cn"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/fe"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/subcontrollers/feproxy"

	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func SetupClusterReconciler(mgr ctrl.Manager) error {
	feController := fe.New(mgr.GetClient(), mgr.GetEventRecorderFor)
	beController := be.New(mgr.GetClient(), mgr.GetEventRecorderFor)
	cnController := cn.New(mgr.GetClient(), mgr.GetEventRecorderFor)
	feProxyController := feproxy.New(mgr.GetClient(), mgr.GetEventRecorderFor)
	subcs := []subcontrollers.ClusterSubController{
		feController, beController, cnController, feProxyController,
	}

	reconciler := &StarRocksClusterReconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("starrockscluster-controller"),
		Scs:      subcs,
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StarRocksClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: AD use secrets instead configmaps
	return ctrl.NewControllerManagedBy(mgr).
		For(&srapi.StarRocksCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&v2.HorizontalPodAutoscaler{}).
		WithEventFilter(predicates.NewGenericPredicates()).
		Complete(r)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StarRocksWarehouseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor("starrockswarehouse-controller")
	r.subControllers = []subcontrollers.WarehouseSubController{cn.New(mgr.GetClient(), mgr.GetEventRecorderFor)}

	// TODO: AD use secrets instead configmaps
	return ctrl.NewControllerManagedBy(mgr).
		For(&srapi.StarRocksWarehouse{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&v2.HorizontalPodAutoscaler{}).
		WithOptions(controller.Options{SkipNameValidation: new(true)}).
		WithEventFilter(predicates.NewGenericPredicates()).
		Complete(r)
}
