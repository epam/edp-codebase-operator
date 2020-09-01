package imagestreamtag

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/imagestreamtag/chain"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("image_stream_tag_controller")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileImageStreamTag{
		client: mgr.GetClient(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("image_stream_tag_controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*edpv1alpha1.ImageStreamTag)
			newObject := e.ObjectNew.(*edpv1alpha1.ImageStreamTag)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	if err := c.Watch(&source.Kind{Type: &edpv1alpha1.ImageStreamTag{}}, &handler.EnqueueRequestForObject{}, p); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileImageStreamTag{}

type ReconcileImageStreamTag struct {
	client client.Client
}

func (r *ReconcileImageStreamTag) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ImageStreamTag")

	ist := &edpv1alpha1.ImageStreamTag{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, ist); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	istChain := chain.CreateDefChain(r.client)
	if err := istChain.ServeRequest(ist); err != nil {
		log.Error(err, err.Error())
		return reconcile.Result{}, err
	}

	log.Info("Reconciling ImageStreamTag has been finished")
	return reconcile.Result{}, nil
}
