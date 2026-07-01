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

package main

import (
	"flag"
	"os"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/controllers"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils"
)

// namespaceList is a repeatable flag.Value that accumulates namespaces, one per occurrence.
type namespaceList []string

func (n *namespaceList) String() string {
	return strings.Join(*n, ",")
}

func (n *namespaceList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*n = append(*n, value)
	return nil
}

var (
	_metricsAddr          string
	_enableLeaderElection bool
	_probeAddr            string

	enableMetricsAuthenticationAndAuthorization bool
	watchNamespaces                             namespaceList
	enableWarehouse                             bool
)

func main() {
	flag.StringVar(&_metricsAddr, "metrics-bind-address", ":8443", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableMetricsAuthenticationAndAuthorization, "metrics-auth", false,
		"Enable metrics authentication and authorization.")
	flag.StringVar(&_probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&_enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Var(&watchNamespaces, "ns", "namespace to watch; "+
		"may be specified multiple times to watch several namespaces. "+
		"If not specified, the manager watches all namespaces.")
	//flag.BoolVar(&config.VolumeNameWithHash, "volume-name-with-hash", true, "Add a hash to the volume name")
	// TODO: AD enabled from values
	flag.BoolVar(&enableWarehouse, "enable-warehouse", false, "Enable the Warehouse controller")

	// Set up logger.
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := ctrl.Log.WithName("main")

	// TODO: AD use standard operator-sdk workflow
	// Register CRD to SchemeBuilder
	srapi.Register()

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   _metricsAddr,
		SecureServing: true,
	}

	if enableMetricsAuthenticationAndAuthorization {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	cacheOptions := cache.Options{
		SyncPeriod: new(2 * time.Minute),
	}
	// If one or more namespaces are specified, restrict the cache to watch objects in those
	// namespaces only. Otherwise, leave DefaultNamespaces nil to watch all namespaces.
	if len(watchNamespaces) > 0 {
		cacheOptions.DefaultNamespaces = make(map[string]cache.Config)
		for _, ns := range watchNamespaces {
			cacheOptions.DefaultNamespaces[ns] = cache.Config{}
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 srapi.Scheme,
		Metrics:                metricsServerOptions,
		Cache:                  cacheOptions,
		HealthProbeBindAddress: _probeAddr,
		LeaderElection:         _enableLeaderElection,
		LeaderElectionID:       "c6c79638.starrocks.com",
	})
	if err != nil {
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// setup all reconciles
	if err = controllers.SetupClusterReconciler(mgr); err != nil {
		logger.Error(err, "unable to set up cluster reconciler")
		os.Exit(1)
	}

	if enableWarehouse {
		if err = (&controllers.StarRocksWarehouseReconciler{}).SetupWithManager(mgr); err != nil {
			logger.Error(err, "unable to create controller", "controller", "Warehouse")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// TODO: AD for what?
	if err = k8sutils.GetKubernetesVersion(); err != nil {
		logger.Error(err, "unable to get kubernetes version, continue to start manager")
	}

	logger.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
}
