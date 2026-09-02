package pod

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/StarRocks/starrocks-kubernetes-operator/cmd/config"
	v1 "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/common"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/common/hash"
)

// SpecialStorageClassName returns the special storage class name of the storage volume, else return "".
// Now we support HostPath and EmptyDir as special storage class.
func SpecialStorageClassName(sv v1.StorageVolume) string {
	storageClassName := sv.StorageClassName
	if storageClassName != nil {
		if common.EqualsIgnoreCase(*storageClassName, v1.EmptyDir) {
			return v1.EmptyDir
		} else if common.EqualsIgnoreCase(*storageClassName, v1.HostPath) {
			return v1.HostPath
		}
		return ""
	}

	if sv.HostPath != nil {
		return v1.HostPath
	}

	return ""
}

// MountStorageVolumes parse StorageVolumes from spec and mount them to pod.
// If StorageClassName is EmptyDir, mount an emptyDir volume to pod.
func MountStorageVolumes(spec v1.SpecInterface) ([]corev1.Volume, []corev1.VolumeMount) {
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	for _, sv := range spec.GetStorageVolumes() {
		if strings.HasPrefix(sv.StorageSize, "0") {
			continue
		}
		switch name := SpecialStorageClassName(sv); name {
		case v1.EmptyDir:
			volumes, volumeMounts = MountEmptyDirVolume(volumes, volumeMounts, sv.Name, sv.MountPath, sv.SubPath)
		case v1.HostPath:
			volumes, volumeMounts = MountHostPathVolume(volumes, volumeMounts, sv.Name, sv.MountPath, sv.SubPath, sv.HostPath)
		default:
			volumes, volumeMounts = MountPersistentVolumeClaim(volumes, volumeMounts, sv.Name, sv.MountPath, sv.SubPath)
		}
	}
	return volumes, volumeMounts
}

func MountPersistentVolumeClaim(volumes []corev1.Volume, volumeMounts []corev1.VolumeMount,
	volumeName, mountPath, subPath string) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes = append(volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volumeName,
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   subPath,
	})
	return volumes, volumeMounts
}

func MountEmptyDirVolume(volumes []corev1.Volume, volumeMounts []corev1.VolumeMount,
	volumeName, mountPath, subPath string) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes = append(volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	volumeMounts = append(
		volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			SubPath:   subPath,
		})
	return volumes, volumeMounts
}

func MountHostPathVolume(volumes []corev1.Volume, volumeMounts []corev1.VolumeMount,
	volumeName string, mountPath string, subPath string,
	hostPath *corev1.HostPathVolumeSource) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes = append(volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: hostPath,
		},
	})
	volumeMounts = append(
		volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			SubPath:   subPath,
		})
	return volumes, volumeMounts
}

// WritableTmpDir is the one path a container with a read-only root filesystem can write to: the start
// scripts put the pid file there and the JVM uses it as java.io.tmpdir.
const WritableTmpDir = "/tmp"

const tmpVolumeName = "starrocks-tmp"

// MountWritableTmpDir mounts an emptyDir at WritableTmpDir when the container runs with a read-only root
// filesystem and the deployment does not mount something there itself.
func MountWritableTmpDir(spec v1.SpecInterface, volumes []corev1.Volume,
	volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {
	if !IsReadOnlyRootFilesystem(spec) {
		return volumes, volumeMounts
	}
	for i := range volumeMounts {
		if volumeMounts[i].MountPath == WritableTmpDir {
			return volumes, volumeMounts
		}
	}
	return MountEmptyDirVolume(volumes, volumeMounts, tmpVolumeName, WritableTmpDir, "")
}

// MountSecrets parse Secrets from spec and mount them to pod.
// The files are mounted with SecretFileMode, so that the credentials a Secret carries are not readable
// by every user of the container.
func MountSecrets(volumes []corev1.Volume, volumeMounts []corev1.VolumeMount,
	references []v1.SecretReference) ([]corev1.Volume, []corev1.VolumeMount) {
	for _, reference := range references {
		volumeName := getVolumeName(v1.MountInfo(reference))
		mode := v1.SecretFileMode
		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  reference.Name,
					DefaultMode: &mode,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: reference.MountPath,
			SubPath:   reference.SubPath,
		})
	}
	return volumes, volumeMounts
}

func getVolumeName(mountInfo v1.MountInfo) string {
	if config.VolumeNameWithHash {
		suffixLen := 4
		suffix := hash.HashObject(mountInfo)
		if len(suffix) > suffixLen {
			suffix = suffix[:suffixLen]
		}
		return mountInfo.Name + "-" + suffix
	}
	return mountInfo.Name
}
