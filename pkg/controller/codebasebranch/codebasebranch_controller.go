package codebasebranch

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/factory"
	cbHandler "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewReconcileCodebaseBranch(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCodebaseBranch {
	return &ReconcileCodebaseBranch{
		client: client,
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
			oo := e.ObjectOld.(*codebaseApi.CodebaseBranch)
			no := e.ObjectNew.(*codebaseApi.CodebaseBranch)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CodebaseBranch{}, builder.WithPredicates(p)).WithOptions(controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
	}).
		Complete(r)
}

func (r *ReconcileCodebaseBranch) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling CodebaseBranch")

	cb := &codebaseApi.CodebaseBranch{}
	if err := r.client.Get(ctx, request.NamespacedName, cb); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer func() {
		if err := r.updateStatus(ctx, cb); err != nil {
			log.Error(err, "error on codebase branch update status")
		}
	}()

	c, err := util.GetCodebase(r.client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.setOwnerRef(cb, c); err != nil {
		setErrorStatus(cb, err.Error())
		return reconcile.Result{}, errors.Wrapf(err, "Unable to set OwnerRef for codebasebranch %v", cb.Name)
	}

	result, err := r.tryToDeleteCodebaseBranch(ctx, cb, factory.GetDeletionChain(c.Spec.CiTool, r.client))
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Unable to remove codebasebranch %v", cb.Name)
	}
	if result != nil {
		return *result, nil
	}

	cbChain := factory.GetChain(c.Spec.CiTool, r.client)
	if err := cbChain.ServeRequest(cb); err != nil {
		log.Error(err, "an error has occurred while handling codebase branch", "name", cb.Name)
		switch err.(type) {
		case *util.CodebaseBranchReconcileError:
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		default:
			return reconcile.Result{}, err
		}
	}

	if err := r.setSuccessStatus(ctx, cb, codebaseApi.JenkinsConfiguration); err != nil {
		return reconcile.Result{},
			errors.Wrapf(err, "an error has been occurred while updating %v Codebase branch status", cb.Name)
	}

	log.Info("Reconciling CodebaseBranch has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) setSuccessStatus(ctx context.Context, cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		LastTimeUpdated:     time.Now(),
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
		if err := r.client.Update(ctx, cb); err != nil {
			return errors.Wrap(err, "ReconcileCodebaseBranch: couldn't update codebase branch status")
		}
	}
	r.log.V(2).Info("codebase branch status has been updated", "name", cb.Name)
	return nil
}

func (r ReconcileCodebaseBranch) tryToDeleteCodebaseBranch(ctx context.Context, cb *codebaseApi.CodebaseBranch,
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
			return err
		}
		r.log.Info("Trying to remove finalizer from", "codenasebranch_name", cb.Name)
		cb.ObjectMeta.Finalizers = util.RemoveString(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName)
		err := r.client.Update(ctx, cb)

		// We have to return err itself here (not wrapped inside another error)
		// so that RetryOnConflict can identify it correctly.
		return err
	})
	if err != nil {
		// May be conflict if max retries were hit, or may be something unrelated
		// like permissions or a network error
		return &reconcile.Result{}, errors.Wrapf(err, "unable to remove finalizer from %v", cb.Name)
	}
	return &reconcile.Result{}, nil
}

func removeDirectoryIfExists(codebaseName, branchName, namespace string) error {
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v", namespace, codebaseName, branchName)
	if !util.DoesDirectoryExist(wd) {
		return nil
	}
	if err := util.RemoveDirectory(wd); err != nil {
		return err
	}
	return nil
}

// setFailureCount increments failure count and returns delay for next reconciliation
func (r *ReconcileCodebaseBranch) setFailureCount(c *codebaseApi.CodebaseBranch) time.Duration {
	timeout := util.GetTimeout(c.Status.FailureCount, 10*time.Second)
	r.log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	c.Status.FailureCount += 1
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
