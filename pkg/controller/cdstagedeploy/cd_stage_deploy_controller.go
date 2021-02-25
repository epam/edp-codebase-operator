package cdstagedeploy

import (
	"context"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	chain "github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/factory"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
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

var log = logf.Log.WithName("cd-stage-deploy-controller")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCDStageDeploy{
		client: mgr.GetClient(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("cd-stage-deploy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.CDStageDeploy)
			on := e.ObjectNew.(*edpv1alpha1.CDStageDeploy)
			if !reflect.DeepEqual(oo.Spec.Tags, on.Spec.Tags) {
				return true
			}
			return false
		},
	}

	if err := c.Watch(&source.Kind{Type: &edpv1alpha1.CDStageDeploy{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCDStageDeploy{}

type ReconcileCDStageDeploy struct {
	client client.Client
}

func (r *ReconcileCDStageDeploy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	vLog := log.WithValues("type", "CDStageDeploy", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	vLog.Info("Reconciling has been started.")

	i := &edpv1alpha1.CDStageDeploy{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client).ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("reconciling has been finished.")
	return reconcile.Result{}, nil
}
