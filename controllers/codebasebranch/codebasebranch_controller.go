package codebasebranch

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	predicateLib "github.com/operator-framework/operator-lib/predicate"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/factory"
	cbHandler "github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func NewReconcileCodebaseBranch(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCodebaseBranch {
	return &ReconcileCodebaseBranch{
		client: c,
		scheme: scheme,
		log:    log.WithName("codebase-branch"),
	}
}

type ReconcileCodebaseBranch struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

const (
	codebaseBranchOperatorFinalizerName = "codebase.branch.operator.finalizer.name"
	errorStatus                         = "error"
)

func (r *ReconcileCodebaseBranch) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*codebaseApi.CodebaseBranch)
			if !ok {
				return false
			}

			no, ok := e.ObjectNew.(*codebaseApi.CodebaseBranch)
			if !ok {
				return false
			}

			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}

			if no.DeletionTimestamp != nil {
				return true
			}

			return false
		},
	}

	pause, err := predicateLib.NewPause(util.PauseAnnotation)
	if err != nil {
		return fmt.Errorf("unable to create pause predicate: %w", err)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CodebaseBranch{}, builder.WithPredicates(p, pause)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build CodebaseBranch controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebasebranches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebasebranches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebasebranches/finalizers,verbs=update

// Reconcile reads that state of the cluster for a CodebaseBranch object and makes changes based on the state.
func (r *ReconcileCodebaseBranch) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling CodebaseBranch")

	cb := &codebaseApi.CodebaseBranch{}
	if err := r.client.Get(ctx, request.NamespacedName, cb); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch CodebaseBranch resource %q: %w", request.NamespacedName, err)
	}

	defer func() {
		if err := r.updateStatus(ctx, cb); err != nil {
			log.Error(err, "error on codebase branch update status")
		}
	}()

	c, err := util.GetCodebase(r.client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch Codebase: %w", err)
	}

	if err = r.setOwnerRef(cb, c); err != nil {
		setErrorStatus(cb, err.Error())
		return reconcile.Result{}, errors.Wrapf(err, "Unable to set OwnerRef for codebasebranch %v", cb.Name)
	}

	if err = codebasebranch.AddCodebaseLabel(ctx, r.client, cb, c.Name); err != nil {
		log.Error(err, "set labels failed")
	}

	result, err := r.tryToDeleteCodebaseBranch(ctx, cb, factory.GetDeletionChain(c.Spec.CiTool, r.client))
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Unable to remove codebasebranch %v", cb.Name)
	}

	if result != nil {
		return *result, nil
	}

	// this is a case where we want to init build number
	// a default build number is a "0"
	// later will be incremented during CI/CD stages
	if c.Spec.Versioning.Type == util.VersioningTypeEDP && cb.Status.Build == nil {
		buildNumber := "0"
		cb.Status.Build = &buildNumber
	}

	cbChain := factory.GetChain(c.Spec.CiTool, r.client)
	if err := cbChain.ServeRequest(cb); err != nil {
		const defaultPostponeTime = 5 * time.Second

		log.Error(err, "an error has occurred while handling codebase branch", "name", cb.Name)

		reconcileErr := util.NewCodebaseBranchReconcileError("error")
		if errors.As(err, &reconcileErr) {
			return reconcile.Result{RequeueAfter: defaultPostponeTime}, nil
		}

		return reconcile.Result{RequeueAfter: r.setFailureCount(cb)}, fmt.Errorf("failed to process default chain: %w", err)
	}

	if err := r.setSuccessStatus(ctx, cb, codebaseApi.CIConfiguration); err != nil {
		return reconcile.Result{},
			errors.Wrapf(err, "an error has been occurred while updating %v Codebase branch status", cb.Name)
	}

	log.Info("Reconciling CodebaseBranch has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) setSuccessStatus(ctx context.Context, cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		LastTimeUpdated:     metaV1.Now(),
		Username:            "system",
		Action:              action,
		Result:              codebaseApi.Success,
		Value:               "active",
		Status:              model.StatusFinished,
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}

	return r.updateStatus(ctx, cb)
}

func (r *ReconcileCodebaseBranch) updateStatus(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	if err := r.client.Status().Update(ctx, cb); err != nil {
		if err != nil {
			r.log.Error(err, "failed to update CodebaseBranch status", "name", cb.Name)
		}

		if err := r.client.Update(ctx, cb); err != nil {
			return errors.Wrap(err, "ReconcileCodebaseBranch: couldn't update codebase branch status")
		}
	}

	r.log.V(2).Info("codebase branch status has been updated", "name", cb.Name)

	return nil
}

func (r *ReconcileCodebaseBranch) tryToDeleteCodebaseBranch(ctx context.Context, cb *codebaseApi.CodebaseBranch,
	deletionChain cbHandler.CodebaseBranchHandler) (*reconcile.Result, error) {
	if cb.GetDeletionTimestamp().IsZero() {
		if !util.ContainsString(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName) {
			cb.ObjectMeta.Finalizers = append(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName)
			if err := r.client.Update(ctx, cb); err != nil {
				return &reconcile.Result{}, errors.Wrapf(err, "unable to add finalizer to %v", cb.Name)
			}
		}

		return nil, nil
	}

	if err := deletionChain.ServeRequest(cb); err != nil {
		switch errors.Cause(err).(type) {
		case service.JobFailedError:
			r.log.Error(err, "deletion job failed")
			return &reconcile.Result{RequeueAfter: r.setFailureCount(cb)}, nil
		default:
			return nil, errors.Wrap(err, "error during deletion chain")
		}
	}

	if err := removeDirectoryIfExists(cb.Spec.CodebaseName, cb.Name, cb.Namespace); err != nil {
		return &reconcile.Result{}, errors.Wrap(err, "unable to remove codebase branch directory")
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the resource here; we need to refetch it on every try, since
		// if we got a conflict on the last update attempt then we need to get
		// the current version before making your own changes.
		if err := r.client.Get(ctx, types.NamespacedName{Namespace: cb.Namespace, Name: cb.Name}, cb); err != nil {
			return err //nolint:wrapcheck // We have to return original error
		}
		r.log.Info("Trying to remove finalizer from", "codebasebranch_name", cb.Name)
		cb.ObjectMeta.Finalizers = util.RemoveString(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName)
		err := r.client.Update(ctx, cb)

		// We have to return err itself here (not wrapped inside another error)
		// so that RetryOnConflict can identify it correctly.
		return err //nolint:wrapcheck // We have to return original error
	})
	if err != nil {
		// May be conflict if max retries were hit, or may be something unrelated
		// like permissions or a network error
		return &reconcile.Result{}, errors.Wrapf(err, "unable to remove finalizer from %v", cb.Name)
	}

	return &reconcile.Result{}, nil
}

func removeDirectoryIfExists(codebaseName, branchName, namespace string) error {
	wd := util.GetWorkDir(codebaseName, fmt.Sprintf("%v-%v", namespace, branchName))
	if err := util.RemoveDirectory(wd); err != nil {
		return fmt.Errorf("failed to remove directory %q: %w", wd, err)
	}

	return nil
}

// setFailureCount increments failure count and returns delay for next reconciliation.
func (r *ReconcileCodebaseBranch) setFailureCount(c *codebaseApi.CodebaseBranch) time.Duration {
	const defaultDuration = 10 * time.Second
	timeout := util.GetTimeout(c.Status.FailureCount, defaultDuration)

	r.log.V(2).Info("wait for next reconciliation", "next reconciliation in", timeout)

	c.Status.FailureCount++

	return timeout
}

func (r *ReconcileCodebaseBranch) setOwnerRef(cb *codebaseApi.CodebaseBranch, c *codebaseApi.Codebase) error {
	if err := controllerutil.SetControllerReference(c, cb, r.scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for CodebaseBranch CR")
	}

	return nil
}

func setErrorStatus(metadata *codebaseApi.CodebaseBranch, msg string) {
	metadata.Status.Status = errorStatus
	metadata.Status.DetailedMessage = msg
}
