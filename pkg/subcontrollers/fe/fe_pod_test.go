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

package fe

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils/templates/pod"
)

const feConfDir = "/opt/starrocks/fe/conf"

func clusterWithFeSpec(feSpec *srapi.StarRocksFeSpec) *srapi.StarRocksCluster {
	feSpec.Image = "starrocks/fe-ubuntu:latest"
	return &srapi.StarRocksCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: srapi.StarRocksClusterSpec{
			StarRocksFeSpec: feSpec,
		},
	}
}

func volumeMountOf(mounts []corev1.VolumeMount, mountPath string) *corev1.VolumeMount {
	for i := range mounts {
		if mounts[i].MountPath == mountPath {
			return &mounts[i]
		}
	}
	return nil
}

func volumeOf(volumes []corev1.Volume, name string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == name {
			return &volumes[i]
		}
	}
	return nil
}

func hasEnv(envs []corev1.EnvVar, name string) bool {
	for i := range envs {
		if envs[i].Name == name {
			return true
		}
	}
	return false
}

// The configuration is mounted where the FE reads it. The entrypoint script of the image does not copy it
// out of a separate directory any more, so CONFIGMAP_MOUNT_PATH is not set either.
func TestBuildPodTemplateMountsTheConfigurationOverConfDir(t *testing.T) {
	t.Run("from a secret", func(t *testing.T) {
		src := clusterWithFeSpec(&srapi.StarRocksFeSpec{
			StarRocksComponentSpec: srapi.StarRocksComponentSpec{
				Secrets: []srapi.SecretReference{{Name: "fe-conf", MountPath: feConfDir}},
			},
		})

		template, err := (&FeController{}).buildPodTemplate(src, nil)
		require.NoError(t, err)

		mount := volumeMountOf(template.Spec.Containers[0].VolumeMounts, feConfDir)
		require.NotNil(t, mount, "the configuration is mounted over %s", feConfDir)
		volume := volumeOf(template.Spec.Volumes, mount.Name)
		require.NotNil(t, volume)
		require.NotNil(t, volume.Secret)
		require.Equal(t, "fe-conf", volume.Secret.SecretName)
		require.NotNil(t, volume.Secret.DefaultMode)
		require.Equal(t, srapi.SecretFileMode, *volume.Secret.DefaultMode,
			"a secret holds credentials and must not be world readable")
		require.False(t, hasEnv(template.Spec.Containers[0].Env, "CONFIGMAP_MOUNT_PATH"))
	})
}

func TestBuildPodTemplateForReadOnlyRootFilesystem(t *testing.T) {
	readOnly := true
	src := clusterWithFeSpec(&srapi.StarRocksFeSpec{
		StarRocksComponentSpec: srapi.StarRocksComponentSpec{
			ReadOnlyRootFilesystem: &readOnly,
		},
	})

	template, err := (&FeController{}).buildPodTemplate(src, nil)
	require.NoError(t, err)

	mount := volumeMountOf(template.Spec.Containers[0].VolumeMounts, pod.WritableTmpDir)
	require.NotNil(t, mount, "a container with a read-only root filesystem needs a writable /tmp")
	volume := volumeOf(template.Spec.Volumes, mount.Name)
	require.NotNil(t, volume)
	require.NotNil(t, volume.EmptyDir)
	require.True(t, hasEnv(template.Spec.Containers[0].Env, "PID_DIR"),
		"FE refuses to start when it can not write its pid file")
}

// FSGroup is what keeps a Secret mounted with SecretFileMode readable for the StarRocks process.
func TestBuildPodTemplateAlwaysSetsFsGroup(t *testing.T) {
	src := clusterWithFeSpec(&srapi.StarRocksFeSpec{})

	template, err := (&FeController{}).buildPodTemplate(src, nil)
	require.NoError(t, err)

	require.NotNil(t, template.Spec.SecurityContext)
	require.NotNil(t, template.Spec.SecurityContext.FSGroup)
	require.Equal(t, srapi.StarRocksGroupID, *template.Spec.SecurityContext.FSGroup)
}
