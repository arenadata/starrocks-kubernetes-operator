package controllers

import (
	"context"
	"testing"

	v1 "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
	"github.com/StarRocks/starrocks-kubernetes-operator/pkg/k8sutils/fake"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func TestSetupClusterReconciler(t *testing.T) {
	type args struct {
		mgr ctrl.Manager
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test setup cluster reconciler",
			args: args{
				mgr: func() ctrl.Manager {
					env := fake.NewEnvironment(fake.WithClusterCRD())
					defer func() {
						err := env.Stop()
						assert.Nil(t, err)
					}()
					return fake.NewManager(env)
				}(),
			},
			wantErr: false,
		},
	}

	v1.Register()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetupClusterReconciler(tt.args.mgr); (err != nil) != tt.wantErr {
				t.Errorf("SetupClusterReconciler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetupWarehouseReconciler(t *testing.T) {
	env1 := fake.NewEnvironment(fake.WithClusterCRD())
	mgr1 := fake.NewManager(env1)
	ctrl.NewControllerManagedBy(mgr1).WithOptions(controller.Options{SkipNameValidation: new(true)})

	env2 := fake.NewEnvironment(fake.WithClusterCRD(), fake.WithWarehouseCRD())
	mgr2 := fake.NewManager(env2)

	defer func() {
		err := env1.Stop()
		assert.Nil(t, err)
		err = env2.Stop()
		assert.Nil(t, err)
	}()

	tests := []struct {
		name    string
		mgr     ctrl.Manager
		wantErr bool
	}{
		{
			name:    "test setup warehouse reconciler with no warehouse CRD",
			mgr:     mgr1,
			wantErr: false,
		},
		{
			name:    "test setup warehouse reconciler with warehouse CRD",
			mgr:     mgr2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := (&StarRocksWarehouseReconciler{}).SetupWithManager(tt.mgr); (err != nil) != tt.wantErr {
				t.Errorf("SetupWarehouseReconciler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type Reader struct {
	hasWarehouseCRD bool
}

func (r Reader) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return nil
}

func (r Reader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	if r.hasWarehouseCRD {
		return nil
	}
	return &meta.NoKindMatchError{}
}

var _ client.Reader = &Reader{}
