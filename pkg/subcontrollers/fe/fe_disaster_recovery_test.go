package fe

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils/fake"
)

func TestShouldEnterDisasterRecoveryMode(t *testing.T) {
	type args struct {
		drSpec   *v1.DisasterRecovery
		drStatus *v1.DisasterRecoveryStatus
		config   map[string]interface{}
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		wantPort int32
	}{
		{
			name: "cluster data is not shared data mode",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: nil,
				config: map[string]interface{}{
					"run_mode": "shared_nothing",
				},
			},
			want: false,
		},
		{
			name: "disaster recovery field is not present",
			args: args{
				drSpec:   nil,
				drStatus: nil,
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want: false,
		},
		{
			name: "disaster recovery is not enabled",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled: false,
				},
				drStatus: nil,
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want: false,
		},
		{
			name: "observed generation equals to generation",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled: true,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					Phase:              v1.DRPhaseDone,
					ObservedGeneration: 0,
				},
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want: false,
		},
		{
			name: "observed generation is larger than generation",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					ObservedGeneration: 2,
				},
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want: false,
		},

		// add test cases that it should enter disaster recovery mode

		{
			name: "fe status is nil",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled: true,
				},
				drStatus: nil,
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want:     true,
			wantPort: 9030,
		},
		{
			name: "generation is larger than observed generation",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					ObservedGeneration: 0,
				},
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want:     true,
			wantPort: 9030,
		},
		{
			name: "generation equal to observed generation, and phase is todo",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					Phase:              v1.DRPhaseTodo,
					ObservedGeneration: 1,
				},
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want:     true,
			wantPort: 9030,
		},
		{
			name: "generation equal to observed generation, and phase is doing",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					Phase:              v1.DRPhaseDoing,
					ObservedGeneration: 1,
				},
				config: map[string]interface{}{
					"run_mode": "shared_data",
				},
			},
			want:     true,
			wantPort: 9030,
		},
		{
			name: "generation equal to observed generation, and phase is doing",
			args: args{
				drSpec: &v1.DisasterRecovery{
					Enabled:    true,
					Generation: 1,
				},
				drStatus: &v1.DisasterRecoveryStatus{
					Phase:              v1.DRPhaseDoing,
					ObservedGeneration: 1,
				},
				config: map[string]interface{}{
					"run_mode":   "shared_data",
					"query_port": "9090",
				},
			},
			want:     true,
			wantPort: 9090,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ShouldEnterDisasterRecoveryMode(tt.args.drSpec, tt.args.drStatus, tt.args.config); got != tt.want {
				t.Errorf("ShouldEnterDisasterRecoveryMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRewriteStatefulSetForDisasterRecovery(t *testing.T) {
	type args struct {
		sts                *appsv1.StatefulSet
		observedGeneration int64
	}
	tests := []struct {
		name string
		args args
		want *appsv1.StatefulSet
	}{
		{
			name: "test rewrite statefulset for disaster recovery",
			args: args{
				sts: &appsv1.StatefulSet{
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										LivenessProbe:  &corev1.Probe{},
										ReadinessProbe: &corev1.Probe{},
										StartupProbe:   &corev1.Probe{},
										Env: []corev1.EnvVar{
											{
												Name:  "key",
												Value: "value",
											},
										},
									},
								},
							},
						},
					},
				},
				observedGeneration: 1,
			},
			want: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: func(i int32) *int32 { return &i }(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									LivenessProbe:  nil,
									ReadinessProbe: PortReadyProbe(9030),
									StartupProbe:   nil,
									Env: []corev1.EnvVar{
										{
											Name:  "key",
											Value: "value",
										},
										{
											Name:  "RESTORE_CLUSTER_GENERATION",
											Value: "1",
										},
										{
											Name:  "RESTORE_CLUSTER_SNAPSHOT",
											Value: "true",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewriteStatefulSetForDisasterRecovery(tt.args.sts, tt.args.observedGeneration, 9030); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rewriteStatefulSetForDisasterRecovery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasClusterSnapshotConf(t *testing.T) {
	confSecret := func(data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "fe-conf", Namespace: "default"},
			Data:       data,
		}
	}
	confDir := "/opt/starrocks/fe/conf"

	tests := []struct {
		name    string
		secrets []v1.SecretReference
		objects []runtime.Object
		want    bool
	}{
		{
			name:    "mounted with a subPath of its own",
			secrets: []v1.SecretReference{{Name: "snapshot", MountPath: confDir + "/cluster_snapshot.yaml", SubPath: "cluster_snapshot.yaml"}},
			want:    true,
		},
		{
			name:    "part of the secret mounted over the conf directory",
			secrets: []v1.SecretReference{{Name: "fe-conf", MountPath: confDir}},
			objects: []runtime.Object{confSecret(map[string][]byte{"fe.conf": []byte(""), "cluster_snapshot.yaml": []byte("")})},
			want:    true,
		},
		{
			// the configuration secret is mounted over the conf directory in every deployment, so the
			// mount path alone must not be taken for a cluster snapshot
			name:    "conf directory mounted without the snapshot",
			secrets: []v1.SecretReference{{Name: "fe-conf", MountPath: confDir}},
			objects: []runtime.Object{confSecret(map[string][]byte{"fe.conf": []byte("")})},
			want:    false,
		},
		{
			name:    "the secret does not exist",
			secrets: []v1.SecretReference{{Name: "fe-conf", MountPath: confDir}},
			want:    false,
		},
		{
			name: "nothing is mounted",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feSpec := &v1.StarRocksFeSpec{
				StarRocksComponentSpec: v1.StarRocksComponentSpec{Secrets: tt.secrets},
			}
			k8sClient := fake.NewFakeClient(scheme.Scheme, tt.objects...)
			got, err := hasClusterSnapshotConf(context.Background(), k8sClient, "default", feSpec)
			if err != nil {
				t.Fatalf("hasClusterSnapshotConf() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("hasClusterSnapshotConf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckFEReadyInDisasterRecovery(t *testing.T) {
	type args struct {
		ctx              context.Context
		k8sClient        client.Client
		clusterNamespace string
		clusterName      string
		generation       int64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// Add cases that FE is not ready
		{
			name: "there is no fe instances",
			args: args{
				ctx:              context.TODO(),
				k8sClient:        fake.NewFakeClient(v1.Scheme),
				clusterNamespace: "default",
				clusterName:      "kube-starrocks",
			},
			want: false,
		},
		{
			name: "there is no container status",
			args: args{
				ctx:              context.TODO(),
				clusterNamespace: "default",
				clusterName:      "kube-starrocks",
				k8sClient: fake.NewFakeClient(v1.Scheme, &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pods",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "kube-starrocks-fe-0",
						Labels: map[string]string{
							v1.ComponentLabelKey: v1.DEFAULT_FE,
							v1.OwnerReference:    "kube-starrocks",
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: nil,
					},
				}),
			},
			want: false,
		},
		{
			name: "container is not ready",
			args: args{
				ctx:              context.TODO(),
				clusterNamespace: "default",
				clusterName:      "kube-starrocks",
				k8sClient: fake.NewFakeClient(v1.Scheme, &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pods",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "kube-starrocks-fe-0",
						Labels: map[string]string{
							v1.ComponentLabelKey: v1.DEFAULT_FE,
							v1.OwnerReference:    "kube-starrocks",
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  v1.DEFAULT_FE,
								Ready: false,
							},
						},
					},
				}),
			},
			want: false,
		},
		{
			name: "environment RESTORE_CLUSTER_GENERATION is not equal to generation",
			args: args{
				ctx:              context.TODO(),
				clusterNamespace: "default",
				clusterName:      "kube-starrocks",
				k8sClient: fake.NewFakeClient(v1.Scheme, &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pods",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "kube-starrocks-fe-0",
						Labels: map[string]string{
							v1.ComponentLabelKey: v1.DEFAULT_FE,
							v1.OwnerReference:    "kube-starrocks",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: v1.DEFAULT_FE,
								Env: []corev1.EnvVar{
									{
										Name:  "RESTORE_CLUSTER_GENERATION",
										Value: "1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  v1.DEFAULT_FE,
								Ready: true,
							},
						},
					},
				}),
				generation: 2,
			},
			want: false,
		},
		// Add cases that FE is ready
		{
			name: "environment RESTORE_CLUSTER_GENERATION is not equal to generation",
			args: args{
				ctx:              context.TODO(),
				clusterNamespace: "default",
				clusterName:      "kube-starrocks",
				k8sClient: fake.NewFakeClient(v1.Scheme, &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pods",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "kube-starrocks-fe-0",
						Labels: map[string]string{
							v1.ComponentLabelKey: v1.DEFAULT_FE,
							v1.OwnerReference:    "kube-starrocks-fe",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: v1.DEFAULT_FE,
								Env: []corev1.EnvVar{
									{
										Name:  "RESTORE_CLUSTER_GENERATION",
										Value: "2",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  v1.DEFAULT_FE,
								Ready: true,
							},
						},
					},
				}),
				generation: 2,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckFEReadyInDisasterRecovery(tt.args.ctx,
				tt.args.k8sClient, tt.args.clusterNamespace, tt.args.clusterName, tt.args.generation); got != tt.want {
				t.Errorf("CheckFEReadyInDisasterRecovery() = %v, want %v", got, tt.want)
			}
		})
	}
}
