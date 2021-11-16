package cdstagedeploy

import (
	"context"
	"fmt"
	"time"

	v1alpha1Stage "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain"
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

func NewReconcileCDStageDeploy(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCDStageDeploy {
	return &ReconcileCDStageDeploy{
		client: client,
		scheme: scheme,
		log:    log.WithName("cd-stage-deploy"),
	}
}

type ReconcileCDStageDeploy struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileCDStageDeploy) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CDStageDeploy{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCDStageDeploy) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("type", "CDStageDeploy", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	log.Info("Reconciling has been started.")

	i := &codebaseApi.CDStageDeploy{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
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
		err := errors.Wrapf(err, "cannot set %v finalizer", util.ForegroundDeletionFinalizerName)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := r.setOwnerRef(ctx, i); err != nil {
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client).ServeRequest(i); err != nil {
		i.SetFailedStatus(err)
		switch err.(type) {
		case *util.CDStageJenkinsDeploymentHasNotBeenProcessed:
			log.Error(err, "unable to continue autodeploy",
				"pipe", i.Spec.Pipeline, "stage", i.Spec.Stage)
			p := r.setReconcilationPeriod(i)
			return reconcile.Result{RequeueAfter: p}, nil
		default:
			return reconcile.Result{}, err
		}
	}

	log.Info("reconciling has been finished.")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageDeploy) setReconcilationPeriod(sd *codebaseApi.CDStageDeploy) time.Duration {
	timeout := util.GetTimeout(sd.Status.FailureCount, 10*time.Second)
	r.log.Info("wait for next reconcilation", "next reconcilation in", timeout)
	sd.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileCDStageDeploy) updateStatus(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	if err := r.client.Status().Update(ctx, stageDeploy); err != nil {
		if err := r.client.Update(ctx, stageDeploy); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageDeploy) setOwnerRef(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	s, err := r.getCDStage(ctx, fmt.Sprintf("%v-%v", stageDeploy.Spec.Pipeline, stageDeploy.Spec.Stage), stageDeploy.Namespace)
	if err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(s, stageDeploy, r.scheme); err != nil {
		return err
	}
	return r.client.Update(context.TODO(), stageDeploy)
}

func (r *ReconcileCDStageDeploy) setFinalizer(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	if !stageDeploy.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if !util.ContainsString(stageDeploy.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName) {
		stageDeploy.ObjectMeta.Finalizers = append(stageDeploy.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName)
	}
	return r.client.Update(ctx, stageDeploy)
}

func (r *ReconcileCDStageDeploy) getCDStage(ctx context.Context, name, namespace string) (*v1alpha1Stage.Stage, error) {
	r.log.Info("getting cd stage", "name", name)
	i := &v1alpha1Stage.Stage{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := r.client.Get(ctx, nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
