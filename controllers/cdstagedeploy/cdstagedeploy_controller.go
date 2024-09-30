package cdstagedeploy

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
)

const requestTimeout = time.Second * 30

func NewReconcileCDStageDeploy(
	c client.Client,
	log logr.Logger,
	chainFactory chain.CDStageDeployChain,
) *ReconcileCDStageDeploy {
	return &ReconcileCDStageDeploy{
		client:       c,
		log:          log.WithName("cd-stage-deploy"),
		chainFactory: chainFactory,
	}
}

type ReconcileCDStageDeploy struct {
	client       client.Client
	log          logr.Logger
	chainFactory chain.CDStageDeployChain
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
			// We need to process only one CDStageDeploy at a time to avoid running multiple pipelines in parallel.
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
//+kubebuilder:rbac:groups=triggers.tekton.dev,namespace=placeholder,resources=triggertemplates,verbs=get;list;watch;
//+kubebuilder:rbac:groups=tekton.dev,namespace=placeholder,resources=pipelineruns,verbs=get;list;watch;create;update;patch

// Reconcile reads that state of the cluster for a CDStageDeploy object and makes changes based on the state.
func (r *ReconcileCDStageDeploy) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling CDStageDeploy has been started")

	stageDeploy := &codebaseApi.CDStageDeploy{}
	if err := r.client.Get(ctx, request.NamespacedName, stageDeploy); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch CDStageDeploy resource %q: %w", request.NamespacedName, err)
	}

	if stageDeploy.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	oldStatus := stageDeploy.Status.Status

	if err := r.chainFactory(r.client, stageDeploy).ServeRequest(ctx, stageDeploy); err != nil {
		stageDeploy.SetFailedStatus(err)

		if statusErr := r.client.Status().Update(ctx, stageDeploy); statusErr != nil {
			log.Error(statusErr, "An error has occurred while updating status field of CDStageDeploy")
		}

		return reconcile.Result{}, fmt.Errorf("failed to process default chainFactory: %w", err)
	}

	if stageDeploy.Status.Status != oldStatus && stageDeploy.Status.Status != codebaseApi.CDStageDeployStatusCompleted {
		if statusErr := r.client.Status().Update(ctx, stageDeploy); statusErr != nil {
			log.Error(statusErr, "An error has occurred while updating status field of CDStageDeploy")
		}
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{
		RequeueAfter: requestTimeout,
	}, nil
}
