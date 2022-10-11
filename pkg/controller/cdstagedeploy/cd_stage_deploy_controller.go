package cdstagedeploy

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func NewReconcileCDStageDeploy(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCDStageDeploy {
	return &ReconcileCDStageDeploy{
		client: c,
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

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CDStageDeploy{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build CDStageDeploy controller: %w", err)
	}

	return nil
}

func (r *ReconcileCDStageDeploy) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("type", "CDStageDeploy", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	log.Info("Reconciling has been started.")

	i := &codebaseApi.CDStageDeploy{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch CDStageDeploy resource %q: %w", request.NamespacedName, err)
	}

	defer func() {
		if err := r.updateStatus(ctx, i); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizer(ctx, i); err != nil {
		err = errors.Wrapf(err, "cannot set %v finalizer", util.ForegroundDeletionFinalizerName)
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
		case *util.CDStageJenkinsDeploymentHasNotBeenProcessedError:
			log.Error(err, "unable to continue autodeploy",
				"pipe", i.Spec.Pipeline, "stage", i.Spec.Stage)

			p := r.setReconciliationPeriod(i)

			return reconcile.Result{RequeueAfter: p}, nil
		default:
			return reconcile.Result{}, fmt.Errorf("failed to process default chain: %w", err)
		}
	}

	log.Info("reconciling has been finished.")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageDeploy) setReconciliationPeriod(sd *codebaseApi.CDStageDeploy) time.Duration {
	const timeoutDuration = 10 * time.Second
	timeout := util.GetTimeout(sd.Status.FailureCount, timeoutDuration)

	r.log.Info("wait for next reconciliation", "next reconciliation in", timeout)

	sd.Status.FailureCount++

	return timeout
}

func (r *ReconcileCDStageDeploy) updateStatus(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	err := r.client.Status().Update(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to update status field of k8s resource: %w", err)
	}

	err = r.client.Update(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to update k8s resource: %w", err)
	}

	return nil
}

func (r *ReconcileCDStageDeploy) setOwnerRef(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	s, err := r.getCDStage(ctx, fmt.Sprintf("%v-%v", stageDeploy.Spec.Pipeline, stageDeploy.Spec.Stage), stageDeploy.Namespace)
	if err != nil {
		return err
	}

	err = controllerutil.SetControllerReference(s, stageDeploy, r.scheme)
	if err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	err = r.client.Update(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to update k8s resource: %w", err)
	}

	return nil
}

func (r *ReconcileCDStageDeploy) setFinalizer(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	if !stageDeploy.GetDeletionTimestamp().IsZero() {
		return nil
	}

	if !util.ContainsString(stageDeploy.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName) {
		stageDeploy.ObjectMeta.Finalizers = append(stageDeploy.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName)
	}

	err := r.client.Update(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to update k8s resource: %w", err)
	}

	return nil
}

func (r *ReconcileCDStageDeploy) getCDStage(ctx context.Context, name, namespace string) (*cdPipeApi.Stage, error) {
	r.log.Info("getting cd stage", "name", name)

	i := &cdPipeApi.Stage{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := r.client.Get(ctx, nn, i); err != nil {
		return nil, fmt.Errorf("failed to fetch k8s resource: %w", err)
	}

	return i, nil
}
