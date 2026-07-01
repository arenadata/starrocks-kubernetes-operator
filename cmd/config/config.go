package config

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	// VolumeNameWithHash decides whether adding a hash to the volume name
	// TODO: AD the resource name is unique in the namespace
	VolumeNameWithHash bool

	// clusterDomain caches the auto-detected cluster domain suffix.
	clusterDomain string
)

//nolint:gochecknoinits
func init() {
	clusterDomain = os.Getenv("K8S_CLUSTER_DOMAIN")
	getClusterDomain()
}

func GetServiceDomainSuffix() string {
	return fmt.Sprintf("svc.%s", getClusterDomain())
}

// getClusterDomain detects the cluster's DNS domain suffix (e.g. "cluster.local") by performing
// a CNAME lookup on the in-cluster apiserver service. The result is cached. On error it falls
// back to "cluster.local".
func getClusterDomain() string {
	if len(clusterDomain) > 0 {
		return clusterDomain
	}

	const defaultDomain = "cluster.local"
	const apiSvc = "kubernetes.default.svc"

	cname, err := net.DefaultResolver.LookupCNAME(context.Background(), apiSvc)
	if err != nil {
		ctrl.Log.WithName("config").Error(err, fmt.Sprintf("failed to lookup %q, falling back to %q", apiSvc, defaultDomain))
		clusterDomain = defaultDomain
		return clusterDomain
	}

	cname = strings.TrimPrefix(cname, apiSvc)
	clusterDomain = strings.Trim(cname, ".")
	if len(clusterDomain) == 0 {
		clusterDomain = defaultDomain
	}

	return clusterDomain
}
