package cdstagedeploy

import (
	"context"
	"fmt"
	v1alpha1Stage "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	chain "github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/factory"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const foregroundDeletionFinalizerName = "foregroundDeletion"

type ReconcileCDStageDeploy struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *ReconcileCDStageDeploy) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CDStageDeploy{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCDStageDeploy) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("type", "CDStageDeploy", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	log.Info("Reconciling has been started.")

	i := &codebaseApi.CDStageDeploy{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defer func() {
		if err := r.updateStatus(ctx, i); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizer(ctx, i); err != nil {
		err := errors.Wrapf(err, "cannot set %v finalizer", foregroundDeletionFinalizerName)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := r.setOwnerReference(ctx, i); err != nil {
		err := errors.Wrapf(err, "cannot set owner ref for %v CDStageDeploy CR", i.Name)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.Client).ServeRequest(i); err != nil {
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}
	i.SetSuccessStatus()

	log.Info("reconciling has been finished.")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageDeploy) updateStatus(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	if err := r.Client.Status().Update(ctx, stageDeploy); err != nil {
		if err := r.Client.Update(ctx, stageDeploy); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageDeploy) setFinalizer(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	if !stageDeploy.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if !util.ContainsString(stageDeploy.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		stageDeploy.ObjectMeta.Finalizers = append(stageDeploy.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
	}
	return r.Client.Update(ctx, stageDeploy)
}

func (r *ReconcileCDStageDeploy) setOwnerReference(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	sn := fmt.Sprintf("%v-%v", stageDeploy.Spec.Pipeline, stageDeploy.Spec.Stage)
	s, err := r.getCDStage(ctx, sn, stageDeploy.Namespace)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(s, stageDeploy, r.Scheme)
}

func (r *ReconcileCDStageDeploy) getCDStage(ctx context.Context, name, namespace string) (*v1alpha1Stage.Stage, error) {
	r.Log.Info("getting cd stage", "name", name)
	i := &v1alpha1Stage.Stage{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := r.Client.Get(ctx, nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
