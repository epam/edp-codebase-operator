package cdstagedeploy

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	chain "github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/factory"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	v1alpha1Stage "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const foregroundDeletionFinalizerName = "foregroundDeletion"

var log = logf.Log.WithName("cd-stage-deploy-controller")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileCDStageDeploy{
		client: mgr.GetClient(),
		scheme: scheme,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	schemeGroupVersionV2 := schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersionV2,
		&v1alpha1Stage.Stage{},
		&v1alpha1Stage.StageList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersionV2)
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("cd-stage-deploy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
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
	scheme *runtime.Scheme
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

	defer func() {
		if err := r.updateStatus(i); err != nil {
			vLog.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizer(i); err != nil {
		err := errors.Wrapf(err, "cannot set %v finalizer", foregroundDeletionFinalizerName)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := r.setOwnerReference(i); err != nil {
		err := errors.Wrapf(err, "cannot set owner ref for %v CDStageDeploy CR", i.Name)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client).ServeRequest(i); err != nil {
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}
	i.SetSuccessStatus()

	vLog.Info("reconciling has been finished.")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageDeploy) updateStatus(stageDeploy *edpv1alpha1.CDStageDeploy) error {
	if err := r.client.Status().Update(context.TODO(), stageDeploy); err != nil {
		if err := r.client.Update(context.TODO(), stageDeploy); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageDeploy) setFinalizer(stageDeploy *edpv1alpha1.CDStageDeploy) error {
	if !stageDeploy.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if !util.ContainsString(stageDeploy.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		stageDeploy.ObjectMeta.Finalizers = append(stageDeploy.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
	}
	return r.client.Update(context.TODO(), stageDeploy)
}

func (r *ReconcileCDStageDeploy) setOwnerReference(stageDeploy *edpv1alpha1.CDStageDeploy) error {
	sn := fmt.Sprintf("%v-%v", stageDeploy.Spec.Pipeline, stageDeploy.Spec.Stage)
	s, err := r.getCDStage(sn, stageDeploy.Namespace)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(s, stageDeploy, r.scheme)
}

func (r *ReconcileCDStageDeploy) getCDStage(name, namespace string) (*v1alpha1Stage.Stage, error) {
	log.Info("getting cd stage", "name", name)
	i := &v1alpha1Stage.Stage{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := r.client.Get(context.TODO(), nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
