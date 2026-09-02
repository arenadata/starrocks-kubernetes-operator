package fe

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils/templates/pod"
)

// GetFEConfig get the fe config from the Secret that is mounted over the conf directory.
// It is not a method of FeController, but BE/CN controller also need to get the fe config.
func GetFEConfig(ctx context.Context, client client.Client,
	feSpec *srapi.StarRocksFeSpec, namespace string) (map[string]interface{}, error) {
	return k8sutils.GetConfig(ctx, client, feSpec.Secrets, pod.GetConfigDir(feSpec), "fe.conf",
		namespace)
}

func IsRunInSharedDataMode(config map[string]interface{}) bool {
	if val := config["run_mode"]; val == nil || !strings.Contains(val.(string), "shared_data") {
		return false
	}
	return true
}
