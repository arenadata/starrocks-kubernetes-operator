package predicates

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// ignoredAnnotation is the annotation key that marks an object as ignored by the operator
	ignoredAnnotation = "starrocks.com/ignored"
)

// GenericPredicates implements predicate.Predicate for filtering events.
type GenericPredicates struct {
	predicate.Funcs
}

// NewGenericPredicates creates a new GenericPredicates.
func NewGenericPredicates() GenericPredicates {
	return GenericPredicates{}
}

// Create returns true if the Create event should be processed
func (gp GenericPredicates) Create(e event.CreateEvent) bool {
	return gp.shouldReconcile(e.Object)
}

// Update returns true if the Update event should be processed
func (gp GenericPredicates) Update(e event.UpdateEvent) bool {
	return gp.shouldReconcile(e.ObjectNew)
}

// Delete returns true if the Delete event should be processed
func (gp GenericPredicates) Delete(e event.DeleteEvent) bool {
	return gp.shouldReconcile(e.Object)
}

// Generic returns true if the Generic event should be processed
func (gp GenericPredicates) Generic(e event.GenericEvent) bool {
	return gp.shouldReconcile(e.Object)
}

// shouldReconcile checks if an object should be reconciled based on annotation filters
func (gp GenericPredicates) shouldReconcile(obj client.Object) bool {
	if obj == nil {
		return false
	}

	return isObjectAllowed(obj)
}

// isObjectAllowed returns true if the object does not have the ignored annotation set to "true"
func isObjectAllowed(obj client.Object) bool {
	if ignoredStatus := obj.GetAnnotations()[ignoredAnnotation]; ignoredStatus == "true" {
		objType := "StarRocks resource"
		if runtimeObj, ok := obj.(runtime.Object); ok {
			objType = fmt.Sprintf("%T", runtimeObj)
		}
		logger := log.Log.WithName("predicates")
		logger.Info("starrocks operator will not reconcile ignored resource, remove annotation to reconcile",
			"type", objType,
			"namespace", obj.GetNamespace(),
			"name", obj.GetName())
		return false
	}
	return true
}
