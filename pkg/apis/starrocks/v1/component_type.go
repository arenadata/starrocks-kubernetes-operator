/*
 * Copyright 2021-present, StarRocks Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"errors"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ SpecInterface = &StarRocksComponentSpec{}

// StarRocksComponentSpec defines the shared specification for all StarRocks components except FE Proxy
type StarRocksComponentSpec struct {
	StarRocksLoadSpec `json:",inline"`

	// SecurityContext holds the security settings of the StarRocks container. Every field set here wins
	// over what the operator derives from the deprecated shortcuts below, everything else keeps its
	// default, so that a deployment can express what its policy requires - a seccomp profile, dropping
	// all capabilities, a specific user - without the operator having to know about the field.
	// See https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// PodSecurityContext holds the security settings of the pod, applied the same way as SecurityContext.
	// Note that the operator sets FSGroup by default: the configuration and the other Secrets are mounted
	// with mode 0440 and belong to that group, without it the StarRocks process can not read them.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// RunAsNonRoot is used to determine whether to run starrocks as a normal user.
	// If RunAsNonRoot is true, operator will set RunAsUser and RunAsGroup to 1000 in securityContext.
	// default: nil
	// Deprecated: set runAsUser/runAsGroup/runAsNonRoot in SecurityContext instead.
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	// refer to https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-capabilities-for-a-container
	// grant certain privileges to a process without granting all the privileges of the root user
	// Deprecated: set capabilities in SecurityContext instead.
	// +optional
	Capabilities *corev1.Capabilities `json:"capabilities,omitempty"`

	// the reference for secrets, which allow users to mount any files to container.
	// +optional
	Secrets []SecretReference `json:"secrets,omitempty"`

	// HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts
	// file if specified. This is only valid for non-hostNetwork pods.
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`

	// TerminationGracePeriodSeconds defines duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
	// Value must be non-negative integer. The value zero indicates stop immediately via
	// the kill signal (no opportunity to shut down).
	// If this value is nil, the default grace period will be used instead.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 120 seconds.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// Sidecars is an optional list of containers that are run in the same pod as the starrocks component.
	// You can use this field to launch helper containers that provide additional functionality to the main container.
	// See https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Container for how to configure a container.
	// +optional
	Sidecars []corev1.Container `json:"sidecars,omitempty"`

	// InitContainers is an optional list of containers that are run in the same pod as the starrocks component.
	// You can use this field to launch helper containers that run before the main container starts.
	// See https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Container for how to configure a container.
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// Entrypoint array. Not executed within a shell.
	// If this is not provided, the ENTRYPOINT of the image (tini) is kept and the default entrypoint script of
	// the component is passed to it as the first argument:
	//	1. For FE, /opt/starrocks/fe_entrypoint.sh.
	//  2. For BE, /opt/starrocks/be_entrypoint.sh.
	//  3. For CN, /opt/starrocks/cn_entrypoint.sh.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint script (or to the command, if it is provided).
	// If this is not provided, it will use $(FE_SERVICE_NAME) for all components.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty"`

	// StarRocksCluster use StatefulSet to deploy FE/BE/CN components.
	// UpdateStrategy indicates the StatefulSetUpdateStrategy that will be
	// employed to update Pods in the StatefulSet when a revision is made to
	// Template. See https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#rolling-updates for more details.
	// Note: The maxUnavailable field is in Alpha stage and it is honored only by API servers that are running with the
	//       MaxUnavailableStatefulSet feature gate enabled.
	// +optional
	UpdateStrategy *appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitempty"`

	// Whether this container has a read-only root filesystem.
	// Default is false.
	// Note that:
	// 	1. This field cannot be set when spec.os.name is windows.
	//	2. The image has to keep its configuration directory read-only and write its pid file outside of
	//     the installation. StarRocks images do so since 4.4.0/4.0.10.1; with an older image the
	//     components fail to start or silently ignore the mounted configuration.
	//	3. When it is enabled the operator mounts an emptyDir at /tmp and points PID_DIR (and
	//     UDF_RUNTIME_DIR for BE/CN) at it. Anything else the deployment writes to - the spill
	//     directory, small files, FE tmp_dir - has to be pointed at a volume in the configuration.
	// Deprecated: set readOnlyRootFilesystem in SecurityContext instead, the operator honors it there
	// as well.
	// +optional
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty" protobuf:"varint,6,opt,name=readOnlyRootFilesystem"`

	// Sysctls defines a list of namespaced sysctls for the podSecurityContext.sysctls
	// See https://kubernetes.io/docs/tasks/administer-cluster/sysctl-cluster/ for more details.
	// Deprecated: set sysctls in PodSecurityContext instead.
	Sysctls []corev1.Sysctl `json:"sysctls,omitempty"`

	// PersistentVolumeClaimRetentionPolicy specifies the retention policy for PersistentVolumeClaims associated with the component.
	// The WhenDeleted field is supported for all components, and it determines whether to delete PVCs when the StatefulSet is deleted.
	// The WhenScaled field is only supported for the CN component.
	//nolint:lll
	PersistentVolumeClaimRetentionPolicy *appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy `json:"persistentVolumeClaimRetentionPolicy,omitempty"`

	// MinReadySeconds specifies the minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready).
	// +optional
	MinReadySeconds *int32 `json:"minReadySeconds,omitempty"`

	// podManagementPolicy controls how pods are created during initial scale up, when replacing pods on nodes, or when scaling down.
	// The default policy is Parallel which is not the same as statefulset's default policy which maybe a not a good choice
	// to use a different default value.
	// +optional
	PodManagementPolicy appsv1.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`
}

// StarRocksComponentStatus represents the status of a starrocks component.
type StarRocksComponentStatus struct {
	// the name of fe service exposed for user.
	ServiceName string `json:"serviceName,omitempty"`

	// FailedInstances failed pod names.
	FailedInstances []string `json:"failedInstances,omitempty"`

	// CreatingInstances in creating pod names.
	CreatingInstances []string `json:"creatingInstances,omitempty"`

	// RunningInstances in running status pod names.
	RunningInstances []string `json:"runningInstances,omitempty"`

	// ResourceNames the statefulset names of fe.
	ResourceNames []string `json:"resourceNames,omitempty"`

	// Phase the value from all pods of component status. If component have one failed pod phase=failed,
	// also if fe have one creating pod phase=creating, also if component all running phase=running, others unknown.
	Phase ComponentPhase `json:"phase"`

	// Reason represents the reason of not running.
	Reason string `json:"reason,omitempty"`
}

type SecretReference MountInfo

const (
	// StarRocksUserID and StarRocksGroupID are the user and the group the StarRocks images create and run
	// with. The operator sets the group as PodSecurityContext.FSGroup for every component, so that the
	// mounted Secrets belong to a group the process is a member of, see SecretFileMode.
	StarRocksUserID  int64 = 1000
	StarRocksGroupID int64 = 1000

	// NginxUserID and NginxGroupID are the user and the group the nginx image of the FE proxy runs with.
	// The operator sets the group as PodSecurityContext.FSGroup of the FE proxy, so that nginx can read
	// its configuration Secret, which is mounted with SecretFileMode.
	NginxUserID  int64 = 101
	NginxGroupID int64 = 101

	// SecretFileMode is the permission the files of a mounted Secret get in the container. Kubernetes
	// defaults to 0644, which makes the credentials a Secret carries - the LDAP bind password, keystore
	// passwords, a Kerberos keytab, and the StarRocks configuration itself - readable by every user of
	// the container. The files belong to root and to the group of PodSecurityContext.FSGroup, which the
	// operator always sets, so 0440 still leaves them readable for the StarRocks process.
	SecretFileMode int32 = 0o440
)

// MountInfo
// The reason why we do not support defaultMode is that we use hash.HashObject to
// calculate the actual volume name. This volume name is used in pod template of statefulset,
// and if this MountInfo type has been changed, the volume name will be changed too, and
// that will make pods restart.
// The permissions of the mounted files are decided by the operator instead, see SecretFileMode.
type MountInfo struct {
	// This must match the Name of a Secret in the same namespace, and
	// the length of name must not more than 50 characters.
	Name string `json:"name,omitempty"`

	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mountPath,omitempty"`

	// SubPath within the volume from which the container's volume should be mounted.
	// Defaults to "" (volume's root).
	// +optional
	SubPath string `json:"subPath,omitempty"`
}

func (spec *StarRocksComponentSpec) GetHostAliases() []corev1.HostAlias {
	return spec.HostAliases
}

func (spec *StarRocksComponentSpec) GetRunAsNonRoot() (*int64, *int64) {
	runAsNonRoot := spec.RunAsNonRoot
	if runAsNonRoot == nil || !*runAsNonRoot {
		return nil, nil
	}

	userID := StarRocksUserID
	groupID := StarRocksGroupID
	return &userID, &groupID
}

func (spec *StarRocksComponentSpec) GetTerminationGracePeriodSeconds() *int64 {
	var defaultSeconds int64 = 120
	if spec.TerminationGracePeriodSeconds == nil {
		return &defaultSeconds
	}
	return spec.TerminationGracePeriodSeconds
}

func (spec *StarRocksComponentSpec) GetCapabilities() *corev1.Capabilities {
	return spec.Capabilities
}

func (spec *StarRocksComponentSpec) GetSidecars() []corev1.Container {
	return spec.Sidecars
}

func (spec *StarRocksComponentSpec) GetInitContainers() []corev1.Container {
	return spec.InitContainers
}

func (spec *StarRocksComponentSpec) GetCommand() []string {
	return spec.Command
}

func (spec *StarRocksComponentSpec) GetArgs() []string {
	return spec.Args
}

func (spec *StarRocksComponentSpec) GetSysctls() []corev1.Sysctl {
	return spec.Sysctls
}

func (spec *StarRocksComponentSpec) GetUpdateStrategy() *appsv1.StatefulSetUpdateStrategy {
	if spec.UpdateStrategy == nil {
		const defaultRollingUpdateStartPod int32 = 0
		return &appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
				Partition: func(v int32) *int32 { return &v }(defaultRollingUpdateStartPod),
			},
		}
	}
	return spec.UpdateStrategy
}

func ValidUpdateStrategy(updateStrategy *appsv1.StatefulSetUpdateStrategy) error {
	if updateStrategy != nil {
		if (updateStrategy.Type == "" || updateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType) &&
			updateStrategy.RollingUpdate != nil {
			rollingUpdate := updateStrategy.RollingUpdate
			if rollingUpdate.MaxUnavailable != nil {
				s := rollingUpdate.MaxUnavailable.String()
				if strings.HasPrefix(s, "0") {
					return errors.New("maxUnavailable field should > 0")
				}
			}
		}
	}
	return nil
}

func (spec *StarRocksComponentSpec) GetSecurityContext() *corev1.SecurityContext {
	return spec.SecurityContext
}

func (spec *StarRocksComponentSpec) GetPodSecurityContext() *corev1.PodSecurityContext {
	return spec.PodSecurityContext
}

func (spec *StarRocksComponentSpec) IsReadOnlyRootFilesystem() *bool {
	if spec.ReadOnlyRootFilesystem == nil {
		b := false
		return &b
	}
	return spec.ReadOnlyRootFilesystem
}

func (spec *StarRocksComponentSpec) GetMinReadySeconds() *int32 {
	return spec.MinReadySeconds
}

func (spec *StarRocksComponentSpec) GetPodManagementPolicy() appsv1.PodManagementPolicyType {
	if spec.PodManagementPolicy == "" {
		return appsv1.ParallelPodManagement
	}
	return spec.PodManagementPolicy
}

// DisasterRecovery is used to determine whether to enter disaster recovery mode.
type DisasterRecovery struct {
	// Enabled is used to determine whether to enter disaster recovery mode.
	Enabled bool `json:"enabled,omitempty"`

	// Generation records the generation of disaster recovery. If you want to trigger disaster recovery, you should
	// increase the generation.
	Generation int64 `json:"generation,omitempty"`
}

// DisasterRecoveryStatus represents the status of disaster recovery.
// Note: you should create a new instance of DisasterRecoveryStatus by NewDisasterRecoveryStatus.
type DisasterRecoveryStatus struct {
	// the available phase include: todo, doing, done
	Phase DRPhase `json:"phase,omitempty"`

	// the reason of disaster recovery.
	Reason string `json:"reason,omitempty"`

	// the unix time of starting disaster recovery.
	StartTimestamp int64 `json:"startTimestamp,omitempty"`

	// the unix time of ending disaster recovery.
	EndTimestamp int64 `json:"endTimestamp,omitempty"`

	// the observed generation of disaster recovery.
	// If the observed generation is less than the generation, it will trigger disaster recovery.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// NewDisasterRecoveryStatus creates a new disaster recovery status which the phase is todo.
func NewDisasterRecoveryStatus(generation int64) *DisasterRecoveryStatus {
	return &DisasterRecoveryStatus{
		Phase:              DRPhaseTodo,
		StartTimestamp:     time.Now().Unix(),
		ObservedGeneration: generation,
	}
}

type DRPhase string

const (
	DRPhaseTodo  DRPhase = "todo"
	DRPhaseDoing DRPhase = "doing"
	DRPhaseDone  DRPhase = "done"
)
