package cdstagedeploy

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func NewReconcileCDStageDeploy(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	chainFactory chain.CDStageDeployChain,
	reconcileDelay int,
) *ReconcileCDStageDeploy {
	return &ReconcileCDStageDeploy{
		client:         c,
		scheme:         scheme,
		log:            log.WithName("cd-stage-deploy"),
		chainFactory:   chainFactory,
		reconcileDelay: reconcileDelay,
	}
}

type ReconcileCDStageDeploy struct {
	client       client.Client
	scheme       *runtime.Scheme
	log          logr.Logger
	chainFactory chain.CDStageDeployChain

	// Waiting time in seconds before starting reconciling.
	reconcileDelay int
}

func (r *ReconcileCDStageDeploy) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*codebaseApi.CDStageDeploy)
			if !ok {
				return false
			}

			no, ok := e.ObjectNew.(*codebaseApi.CDStageDeploy)
			if !ok {
				return false
			}

			if codebasepredicate.PauseAnnotationChanged(oo, no) {
				return true
			}

			return false
		},
	}

	pause := codebasepredicate.NewPause(r.log)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CDStageDeploy{}, builder.WithPredicates(p, pause)).
		WithOptions(controller.Options{
			// We need to process only one CDStageDeploy at a time to support reconcileDelay.
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build CDStageDeploy controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdstagedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdstagedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdstagedeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=argoproj.io,namespace=placeholder,resources=applications,verbs=get;list;watch;update;patch;
//+kubebuilder:rbac:groups=triggers.tekton.dev,namespace=placeholder,resources=triggertemplates,verbs=get;list;watch;
//+kubebuilder:rbac:groups=tekton.dev,namespace=placeholder,resources=pipelineruns,verbs=create;

// Reconcile reads that state of the cluster for a CDStageDeploy object and makes changes based on the state.
func (r *ReconcileCDStageDeploy) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	if !r.wait(ctx) {
		return reconcile.Result{}, nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling CDStageDeploy has been started")

	i := &codebaseApi.CDStageDeploy{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch CDStageDeploy resource %q: %w", request.NamespacedName, err)
	}

	defer func() {
		if err := r.updateStatus(ctx, i); err != nil {
			log.Error(err, "Error during status updating")
		}
	}()

	if err := r.setFinalizer(ctx, i); err != nil {
		err = fmt.Errorf("failed to set %v finalizer: %w", util.ForegroundDeletionFinalizerName, err)
		i.SetFailedStatus(err)

		return reconcile.Result{}, err
	}

	if err := r.setOwnerRef(ctx, i); err != nil {
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	ch, err := r.chainFactory(r.client)
	if err != nil {
		i.SetFailedStatus(err)

		return reconcile.Result{}, err
	}

	if err = ch.ServeRequest(ctx, i); err != nil {
		i.SetFailedStatus(err)

		return reconcile.Result{}, fmt.Errorf("failed to process default chainFactory: %w", err)
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{}, nil
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

	if controllerutil.AddFinalizer(stageDeploy, util.ForegroundDeletionFinalizerName) {
		err := r.client.Update(ctx, stageDeploy)
		if err != nil {
			return fmt.Errorf("failed to update k8s resource: %w", err)
		}
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

// wait for a specified time before starting reconciling.
// If the context is canceled, the function returns false.
func (r *ReconcileCDStageDeploy) wait(ctx context.Context) bool {
	w := time.After(time.Second * time.Duration(r.reconcileDelay))
	l := ctrl.LoggerFrom(ctx)

	l.Info(fmt.Sprintf("Waiting for %d seconds to start reconciling.", r.reconcileDelay))

	for {
		select {
		case <-ctx.Done():
			l.Info("CDStageDeploy reconciling has been stopped")
			return false
		case <-w:
			return true
		}
	}
}
