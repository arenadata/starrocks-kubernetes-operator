// Copyright 2021-present, StarRocks Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pod

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	v1 "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	rutils "github.com/StarRocks/starrocks-kubernetes-operator/pkg/common/resource_utils"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils/templates/service"
)

func TestLabels(t *testing.T) {
	type args struct {
		clusterName string
		spec        v1.SpecInterface
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test labels",
			args: args{
				clusterName: "test",
				spec: &v1.StarRocksFeSpec{
					StarRocksComponentSpec: v1.StarRocksComponentSpec{
						StarRocksLoadSpec: v1.StarRocksLoadSpec{
							PodLabels: map[string]string{
								"l1": "v1",
							},
						},
					},
				},
			},
			want: map[string]string{
				"l1":                 "v1",
				v1.OwnerReference:    "test-fe",
				v1.ComponentLabelKey: v1.DEFAULT_FE,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Labels(tt.args.clusterName, tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Labels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvs(t *testing.T) {
	envsWithoutIP := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
		{
			Name:  "HOST_TYPE",
			Value: "FQDN",
		},
	}

	envs := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		},
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"},
			},
		},
		{
			Name: "HOST_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
		{
			Name:  "HOST_TYPE",
			Value: "FQDN",
		},
	}

	type args struct {
		clusterName string
		namespace   string
		spec        v1.SpecInterface
		config      map[string]interface{}
	}
	tests := []struct {
		name            string
		args            args
		want            []corev1.EnvVar
		unsupportedEnvs string
	}{
		{
			name: "test envs for fe",
			args: args{
				clusterName: "test",
				namespace:   "ns",
				spec:        &v1.StarRocksFeSpec{},
			},
			want: append(append([]corev1.EnvVar(nil), envs...), []corev1.EnvVar{
				{
					Name:  v1.COMPONENT_NAME,
					Value: v1.DEFAULT_FE,
				},
				{
					Name:  v1.FE_SERVICE_NAME,
					Value: service.ExternalServiceName("test", &v1.StarRocksFeSpec{}) + "." + "ns",
				},
			}...),
			unsupportedEnvs: "",
		},
		{
			name: "test envs for be",
			args: args{
				clusterName: "test",
				namespace:   "ns",
				spec:        &v1.StarRocksBeSpec{},
			},
			want: append(append([]corev1.EnvVar(nil), envs...), []corev1.EnvVar{
				{
					Name:  v1.COMPONENT_NAME,
					Value: v1.DEFAULT_BE,
				},
				{
					Name:  v1.FE_SERVICE_NAME,
					Value: service.ExternalServiceName("test", &v1.StarRocksFeSpec{}),
				},
				{
					Name:  "FE_QUERY_PORT",
					Value: fmt.Sprintf("%v", rutils.DefMap[rutils.QUERY_PORT]),
				},
			}...),
			unsupportedEnvs: "",
		},
		{
			name: "test envs for cn",
			args: args{
				clusterName: "test",
				namespace:   "ns",
				spec:        &v1.StarRocksCnSpec{},
			},
			want: append(append([]corev1.EnvVar(nil), envs...), []corev1.EnvVar{
				{
					Name:  v1.COMPONENT_NAME,
					Value: v1.DEFAULT_CN,
				},
				{
					Name:  v1.FE_SERVICE_NAME,
					Value: service.ExternalServiceName("test", &v1.StarRocksFeSpec{}),
				},
				{
					Name:  "FE_QUERY_PORT",
					Value: fmt.Sprintf("%v", rutils.DefMap[rutils.QUERY_PORT]),
				},
			}...),
			unsupportedEnvs: "",
		},
		{
			name: "test envs for be with unsupport envs",
			args: args{
				clusterName: "test",
				namespace:   "ns",
				spec:        &v1.StarRocksBeSpec{},
			},
			want: append(append([]corev1.EnvVar(nil), envsWithoutIP...), []corev1.EnvVar{
				{
					Name:  v1.COMPONENT_NAME,
					Value: v1.DEFAULT_BE,
				},
				{
					Name:  v1.FE_SERVICE_NAME,
					Value: service.ExternalServiceName("test", &v1.StarRocksFeSpec{}),
				},
				{
					Name:  "FE_QUERY_PORT",
					Value: fmt.Sprintf("%v", rutils.DefMap[rutils.QUERY_PORT]),
				},
			}...),
			unsupportedEnvs: "HOST_IP,POD_IP",
		},
	}
	for _, tt := range tests {
		feExternalServiceName := service.ExternalServiceName("test", &v1.StarRocksFeSpec{})
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("KUBE_STARROCKS_UNSUPPORTED_ENVS", tt.unsupportedEnvs)
			defer func() {
				os.Setenv("KUBE_STARROCKS_UNSUPPORTED_ENVS", "")
			}()
			got := Envs(tt.args.spec, tt.args.config, feExternalServiceName, tt.args.namespace, nil)
			if len(got) != len(tt.want) {
				t.Errorf("Envs() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i].ValueFrom != nil {
					if got[i].ValueFrom.FieldRef.FieldPath != tt.want[i].ValueFrom.FieldRef.FieldPath {
						t.Errorf("Envs() = %v, want %v", got[i], tt.want[i])
					}
				} else if got[i] != tt.want[i] {
					t.Errorf("Envs() = %v, want %v", got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSpec(t *testing.T) {
	type args struct {
		spec      v1.SpecInterface
		container corev1.Container
		volumes   []corev1.Volume
	}
	tests := []struct {
		name string
		args args
		want corev1.PodSpec
	}{
		{
			name: "test service account name in spec",
			args: args{
				spec: &v1.StarRocksFeSpec{
					StarRocksComponentSpec: v1.StarRocksComponentSpec{
						StarRocksLoadSpec: v1.StarRocksLoadSpec{
							ServiceAccount: "test",
						},
					},
				},
				container: corev1.Container{},
				volumes:   nil,
			},
			want: corev1.PodSpec{
				Containers:                    []corev1.Container{{}},
				ServiceAccountName:            "test",
				TerminationGracePeriodSeconds: rutils.GetInt64ptr(int64(120)),
				AutomountServiceAccountToken:  func() *bool { b := false; return &b }(),
			},
		},
		{
			name: "test service account name 2 in spec",
			args: args{
				spec:      &v1.StarRocksFeSpec{},
				container: corev1.Container{},
				volumes:   nil,
			},
			want: corev1.PodSpec{
				Containers:                    []corev1.Container{{}},
				TerminationGracePeriodSeconds: rutils.GetInt64ptr(int64(120)),
				AutomountServiceAccountToken:  func() *bool { b := false; return &b }(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Spec(tt.args.spec, tt.args.container, tt.args.volumes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Spec() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSecurityContext(t *testing.T) {
	onrootMismatch := corev1.FSGroupChangeOnRootMismatch
	// FSGroup is set for every component, so that the files of a Secret mounted with SecretFileMode
	// belong to a group the StarRocks process is a member of.
	fsGroup := v1.StarRocksGroupID

	type args struct {
		spec v1.SpecInterface
	}
	tests := []struct {
		name string
		args args
		want *corev1.PodSecurityContext
	}{
		{
			name: "test security context",
			args: args{
				spec: &v1.StarRocksFeSpec{
					StarRocksComponentSpec: v1.StarRocksComponentSpec{},
				},
			},
			want: &corev1.PodSecurityContext{
				FSGroupChangePolicy: &onrootMismatch,
				FSGroup:             &fsGroup,
			},
		},
		{
			name: "test security context 2",
			args: args{
				spec: &v1.StarRocksFeSpec{},
			},
			want: &corev1.PodSecurityContext{
				FSGroupChangePolicy: &onrootMismatch,
				FSGroup:             &fsGroup,
			},
		},
		{
			name: "run as non root keeps its own group",
			args: args{
				spec: &v1.StarRocksFeSpec{
					StarRocksComponentSpec: v1.StarRocksComponentSpec{
						RunAsNonRoot: func() *bool { b := true; return &b }(),
					},
				},
			},
			want: &corev1.PodSecurityContext{
				FSGroupChangePolicy: &onrootMismatch,
				FSGroup:             &fsGroup,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PodSecurityContext(tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodSecurityContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnnotations(t *testing.T) {
	type args struct {
		spec v1.SpecInterface
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test annotations",
			args: args{
				spec: &v1.StarRocksFeSpec{
					StarRocksComponentSpec: v1.StarRocksComponentSpec{
						StarRocksLoadSpec: v1.StarRocksLoadSpec{
							Annotations: map[string]string{"v1": "v1"},
						},
					},
				},
			},
			want: map[string]string{
				"v1": "v1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Annotations(tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Annotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerSecurityContext(t *testing.T) {
	falseFlag := false
	falseFlagPtr := &falseFlag
	addCapabilities := corev1.Capabilities{
		Add: []corev1.Capability{"SYS_PTRACE", "PERFMON"},
	}
	addDropCapabilities := corev1.Capabilities{
		Add:  []corev1.Capability{"SYS_PTRACE", "PERFMON"},
		Drop: []corev1.Capability{"SYS_ADMIN"},
	}

	type args struct {
		spec v1.SpecInterface
	}
	tests := []struct {
		name string
		args args
		want *corev1.SecurityContext
	}{
		{
			name: "no capabilities",
			args: args{
				spec: &v1.StarRocksComponentSpec{
					RunAsNonRoot: nil,
					Capabilities: nil,
				},
			},
			want: &corev1.SecurityContext{
				ReadOnlyRootFilesystem:   falseFlagPtr,
				AllowPrivilegeEscalation: falseFlagPtr,
				Capabilities:             nil,
			},
		},
		{
			name: "add capabilities",
			args: args{
				spec: &v1.StarRocksComponentSpec{
					RunAsNonRoot: nil,
					Capabilities: &addCapabilities,
				},
			},
			want: &corev1.SecurityContext{
				ReadOnlyRootFilesystem:   falseFlagPtr,
				AllowPrivilegeEscalation: falseFlagPtr,
				Capabilities:             &addCapabilities,
			},
		},
		{
			name: "add/drop capabilities",
			args: args{
				spec: &v1.StarRocksComponentSpec{
					RunAsNonRoot: nil,
					Capabilities: &addDropCapabilities,
				},
			},
			want: &corev1.SecurityContext{
				ReadOnlyRootFilesystem:   falseFlagPtr,
				AllowPrivilegeEscalation: falseFlagPtr,
				Capabilities:             &addDropCapabilities,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainerSecurityContext(tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ContainerSecurityContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestK8sYamlMarshal(t *testing.T) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "input is map",
			args: args{
				data: map[string]string{
					"hello": `metadata:
  annotations:
    app.starrocks.io/fe-config-hash: e615d940
  creationTimestamp: "2023-12-13T05:25:55Z"
`,
				},
			},
			want: `hello: |
  metadata:
    annotations:
      app.starrocks.io/fe-config-hash: e615d940
    creationTimestamp: "2023-12-13T05:25:55Z"
`,
		},
		{
			name: "input is slice",
			args: args{
				data: []string{
					`metadata:
  annotations:
    app.starrocks.io/fe-config-hash: e615d940
  creationTimestamp: "2023-12-13T05:25:55Z"
`,
				},
			},
			want: `- |
  metadata:
    annotations:
      app.starrocks.io/fe-config-hash: e615d940
    creationTimestamp: "2023-12-13T05:25:55Z"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := yaml.Marshal(tt.args.data); err != nil {
				t.Errorf("yaml marshal failed: %v", err)
			} else if string(got) != tt.want {
				t.Errorf("yaml marshal not expected, got: \n%v\n\nwant: \n%v\n", string(got), tt.want)
			}
		})
	}
}

func TestGetStarRocksRootPath(t *testing.T) {
	type args struct {
		envVars []corev1.EnvVar
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test get starrocks root path - 1",
			args: args{
				envVars: nil,
			},
			want: "/opt/starrocks",
		},
		{
			name: "test get starrocks root path - 2",
			args: args{
				envVars: []corev1.EnvVar{
					{
						Name:  "STARROCKS_ROOT",
						Value: "xxx",
					},
				},
			},
			want: "xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStarRocksRootPath(tt.args.envVars); got != tt.want {
				t.Errorf("GetStarRocksRootPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerCommandAndArgs(t *testing.T) {
	tests := []struct {
		name        string
		spec        v1.SpecInterface
		wantCommand []string
		wantArgs    []string
	}{
		{
			name:        "default: keep the image entrypoint, pass the script through args",
			spec:        &v1.StarRocksFeSpec{},
			wantCommand: nil,
			wantArgs:    []string{"/opt/starrocks/fe_entrypoint.sh", "$(FE_SERVICE_NAME)"},
		},
		{
			name: "custom root path",
			spec: &v1.StarRocksBeSpec{
				BeEnvVars: []corev1.EnvVar{{Name: "STARROCKS_ROOT", Value: "/data/starrocks"}},
			},
			wantCommand: nil,
			wantArgs:    []string{"/data/starrocks/be_entrypoint.sh", "$(FE_SERVICE_NAME)"},
		},
		{
			name: "user args are passed to the default entrypoint script",
			spec: &v1.StarRocksCnSpec{
				StarRocksComponentSpec: v1.StarRocksComponentSpec{Args: []string{"my-fe-svc"}},
			},
			wantCommand: nil,
			wantArgs:    []string{"/opt/starrocks/cn_entrypoint.sh", "my-fe-svc"},
		},
		{
			name: "user command replaces the entrypoint, default args are kept",
			spec: &v1.StarRocksFeSpec{
				StarRocksComponentSpec: v1.StarRocksComponentSpec{Command: []string{"/my/entrypoint.sh"}},
			},
			wantCommand: []string{"/my/entrypoint.sh"},
			wantArgs:    []string{"$(FE_SERVICE_NAME)"},
		},
		{
			name: "user command and args are passed as they are",
			spec: &v1.StarRocksFeSpec{
				StarRocksComponentSpec: v1.StarRocksComponentSpec{
					Command: []string{"bash", "-c"},
					Args:    []string{"/my/entrypoint.sh $(FE_SERVICE_NAME)"},
				},
			},
			wantCommand: []string{"bash", "-c"},
			wantArgs:    []string{"/my/entrypoint.sh $(FE_SERVICE_NAME)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainerCommand(tt.spec); !reflect.DeepEqual(got, tt.wantCommand) {
				t.Errorf("ContainerCommand() = %v, want %v", got, tt.wantCommand)
			}
			if got := ContainerArgs(tt.spec); !reflect.DeepEqual(got, tt.wantArgs) {
				t.Errorf("ContainerArgs() = %v, want %v", got, tt.wantArgs)
			}
		})
	}
}

func envValue(envs []corev1.EnvVar, name string) (string, bool) {
	for i := range envs {
		if envs[i].Name == name {
			return envs[i].Value, true
		}
	}
	return "", false
}

// With a read-only root filesystem the pid file and the runtime directory of the UDFs can not stay in the
// installation directory, they go to the emptyDir mounted at /tmp.
func TestEnvsForReadOnlyRootFilesystem(t *testing.T) {
	readOnly := func(value bool) *bool { return &value }

	t.Run("fe writes its pid file to the writable directory", func(t *testing.T) {
		spec := &v1.StarRocksFeSpec{
			StarRocksComponentSpec: v1.StarRocksComponentSpec{ReadOnlyRootFilesystem: readOnly(true)},
		}

		envs := Envs(spec, map[string]interface{}{}, "fe-service", "default", nil)

		pidDir, ok := envValue(envs, "PID_DIR")
		require.True(t, ok)
		require.Equal(t, WritableTmpDir, pidDir)
		_, ok = envValue(envs, "UDF_RUNTIME_DIR")
		require.False(t, ok, "FE has no UDF runtime directory")
	})

	t.Run("be also gets a writable udf runtime directory", func(t *testing.T) {
		spec := &v1.StarRocksBeSpec{
			StarRocksComponentSpec: v1.StarRocksComponentSpec{ReadOnlyRootFilesystem: readOnly(true)},
		}

		envs := Envs(spec, map[string]interface{}{}, "fe-service", "default", nil)

		pidDir, ok := envValue(envs, "PID_DIR")
		require.True(t, ok)
		require.Equal(t, WritableTmpDir, pidDir)
		udfDir, ok := envValue(envs, "UDF_RUNTIME_DIR")
		require.True(t, ok)
		require.Equal(t, WritableTmpDir+"/udf-runtime", udfDir)
	})

	t.Run("nothing is added for a writable root filesystem", func(t *testing.T) {
		spec := &v1.StarRocksBeSpec{
			StarRocksComponentSpec: v1.StarRocksComponentSpec{ReadOnlyRootFilesystem: readOnly(false)},
		}

		envs := Envs(spec, map[string]interface{}{}, "fe-service", "default", nil)

		_, ok := envValue(envs, "PID_DIR")
		require.False(t, ok)
		_, ok = envValue(envs, "UDF_RUNTIME_DIR")
		require.False(t, ok)
	})

	t.Run("the deployment can point the pid file somewhere else", func(t *testing.T) {
		spec := &v1.StarRocksFeSpec{
			StarRocksComponentSpec: v1.StarRocksComponentSpec{ReadOnlyRootFilesystem: readOnly(true)},
		}

		envs := Envs(spec, map[string]interface{}{}, "fe-service", "default",
			[]corev1.EnvVar{{Name: "PID_DIR", Value: "/var/run/starrocks"}})

		pidDir, _ := envValue(envs, "PID_DIR")
		require.Equal(t, "/var/run/starrocks", pidDir)
	})
}
