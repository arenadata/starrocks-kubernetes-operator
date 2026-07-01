package predicates

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	srapi "github.com/StarRocks/starrocks-kubernetes-operator/pkg/apis/starrocks/v1"
)

func TestGenericPredicates_Create(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		annotations map[string]string
		want        bool
	}{
		{
			name:      "allow object without annotation",
			namespace: "default",
			want:      true,
		},
		{
			name:        "deny object with ignored annotation",
			namespace:   "default",
			annotations: map[string]string{ignoredAnnotation: "true"},
			want:        false,
		},
		{
			name:        "allow object with ignored annotation set to false",
			namespace:   "default",
			annotations: map[string]string{ignoredAnnotation: "false"},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test object
			obj := &srapi.StarRocksCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cluster",
					Namespace:   tt.namespace,
					Annotations: tt.annotations,
				},
			}

			// Create event
			e := event.CreateEvent{
				Object: obj,
			}

			// Test predicate with constructor
			gp := NewGenericPredicates()
			if got := gp.Create(e); got != tt.want {
				t.Errorf("GenericPredicates.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenericPredicates_Update(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		annotations map[string]string
		want        bool
	}{
		{
			name:      "allow update without annotation",
			namespace: "default",
			want:      true,
		},
		{
			name:        "deny update with ignored annotation",
			namespace:   "default",
			annotations: map[string]string{ignoredAnnotation: "true"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test objects
			oldObj := &srapi.StarRocksCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: tt.namespace,
				},
			}
			newObj := &srapi.StarRocksCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cluster",
					Namespace:   tt.namespace,
					Annotations: tt.annotations,
				},
			}

			// Create event
			e := event.UpdateEvent{
				ObjectOld: oldObj,
				ObjectNew: newObj,
			}

			// Test predicate with constructor
			gp := NewGenericPredicates()
			if got := gp.Update(e); got != tt.want {
				t.Errorf("GenericPredicates.Update() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenericPredicates_Delete(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		annotations map[string]string
		want        bool
	}{
		{
			name:      "allow delete without annotation",
			namespace: "default",
			want:      true,
		},
		{
			name:        "deny delete with ignored annotation",
			namespace:   "default",
			annotations: map[string]string{ignoredAnnotation: "true"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &srapi.StarRocksCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cluster",
					Namespace:   tt.namespace,
					Annotations: tt.annotations,
				},
			}

			e := event.DeleteEvent{
				Object: obj,
			}

			gp := NewGenericPredicates()
			if got := gp.Delete(e); got != tt.want {
				t.Errorf("GenericPredicates.Delete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenericPredicates_Generic(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		annotations map[string]string
		want        bool
	}{
		{
			name:      "allow generic without annotation",
			namespace: "default",
			want:      true,
		},
		{
			name:        "deny generic with ignored annotation",
			namespace:   "default",
			annotations: map[string]string{ignoredAnnotation: "true"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &srapi.StarRocksCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cluster",
					Namespace:   tt.namespace,
					Annotations: tt.annotations,
				},
			}

			e := event.GenericEvent{
				Object: obj,
			}

			gp := NewGenericPredicates()
			if got := gp.Generic(e); got != tt.want {
				t.Errorf("GenericPredicates.Generic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldReconcile_WithNilObject(t *testing.T) {
	// Test that nil objects are handled gracefully
	gp := NewGenericPredicates()

	// Test Create with nil object
	e1 := event.CreateEvent{Object: nil}
	if got := gp.Create(e1); got != false {
		t.Errorf("GenericPredicates.Create() with nil object = %v, want false", got)
	}

	// Test Delete with nil object
	e2 := event.DeleteEvent{Object: nil}
	if got := gp.Delete(e2); got != false {
		t.Errorf("GenericPredicates.Delete() with nil object = %v, want false", got)
	}

	// Test Generic with nil object
	e3 := event.GenericEvent{Object: nil}
	if got := gp.Generic(e3); got != false {
		t.Errorf("GenericPredicates.Generic() with nil object = %v, want false", got)
	}
}
