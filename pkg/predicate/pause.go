package predicate

import (
	"strconv"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// pauseAnnotation is a key for pause annotation that disables custom resource reconciliation.
const pauseAnnotation = "edp.epam.com/paused"

// Pause is a predicate that checks if a custom resource has a pause annotation.
// The operator will not reconcile the resource if the annotation is present and set to true.
type Pause struct {
	annotation string
	log        logr.Logger
}

// NewPause returns a new Pause predicate.
func NewPause(log logr.Logger) *Pause {
	return &Pause{annotation: pauseAnnotation, log: log.WithName("pause-predicate")}
}

// Create returns true if the object has no pause annotation or the annotation is set to false.
func (p Pause) Create(evt event.CreateEvent) bool {
	if evt.Object == nil {
		p.log.Info("Create evt object is nil")

		return false
	}

	return p.run(evt.Object)
}

// Delete returns true if the object has no pause annotation or the annotation is set to false.
func (p Pause) Delete(evt event.DeleteEvent) bool {
	if evt.Object == nil {
		p.log.Info("Delete evt object is nil")

		return false
	}

	return p.run(evt.Object)
}

// Update returns true if the object has no pause annotation or the annotation is set to false.
func (p Pause) Update(evt event.UpdateEvent) bool {
	if evt.ObjectNew != nil {
		return p.run(evt.ObjectNew)
	}

	if evt.ObjectOld != nil {
		return p.run(evt.ObjectOld)
	}

	p.log.Info("Update evt objects are nil")

	return false
}

// Generic returns true if the object has no pause annotation or the annotation is set to false.
func (p Pause) Generic(evt event.GenericEvent) bool {
	if evt.Object == nil {
		p.log.Info("Generic evt object is nil")

		return false
	}

	return p.run(evt.Object)
}

// run checks if the object has a pause annotation and returns true if the annotation is not present or set to false.
func (p Pause) run(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	if len(annotations) == 0 {
		return true
	}

	annoStr, hasAnno := annotations[p.annotation]
	if !hasAnno {
		return true
	}

	pause, err := strconv.ParseBool(annoStr)
	if err != nil {
		return true
	}

	if pause {
		p.log.Info(
			"Resource reconciliation is paused",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
	}

	return !pause
}

// PauseAnnotationChanged returns true if the pause annotation has been changed.
func PauseAnnotationChanged(objOld, objNew client.Object) bool {
	if objOld == nil || objNew == nil {
		return false
	}

	oldAnno := objOld.GetAnnotations()
	newAnno := objNew.GetAnnotations()

	oldStr, oldHasAnno := oldAnno[pauseAnnotation]
	newStr, newHasAnno := newAnno[pauseAnnotation]

	if oldHasAnno && !newHasAnno {
		return true
	}

	if !oldHasAnno && newHasAnno {
		return true
	}

	return oldStr != newStr
}
