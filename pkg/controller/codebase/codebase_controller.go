package codebase

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-codebase-operator/v2/db"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain"
	cHand "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	validate "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/validation"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const codebaseOperatorFinalizerName = "codebase.operator.finalizer.name"

func NewReconcileCodebase(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCodebase {
	return &ReconcileCodebase{
		client: client,
		scheme: scheme,
		db:     db.GetConnection(),
		log:    log.WithName("codebase"),
	}
}

type ReconcileCodebase struct {
	client      client.Client
	scheme      *runtime.Scheme
	db          *sql.DB
	log         logr.Logger
	chainGetter func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error)
}

func (r *ReconcileCodebase) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*codebaseApi.Codebase)
			no := e.ObjectNew.(*codebaseApi.Codebase)
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
		For(&codebaseApi.Codebase{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCodebase) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Codebase")

	c := &codebaseApi.Codebase{}
	if err := r.client.Get(ctx, request.NamespacedName, c); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer func() {
		if err := r.updateStatus(ctx, c); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizers(ctx, c); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "unable to set finalizers")
	}

	result, err := r.tryToDeleteCodebase(ctx, c)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while trying to delete codebase")
	}
	if result != nil {
		return *result, nil
	}

	if !validate.IsCodebaseValid(c) {
		return reconcile.Result{}, nil
	}
	ch, err := r.getChain(c)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := ch.ServeRequest(c); err != nil {
		if pErr, ok := errors.Cause(err).(chain.PostponeError); ok {
			return reconcile.Result{RequeueAfter: pErr.Timeout}, nil
		}

		timeout := r.setFailureCount(c)
		log.Error(err, "an error has occurred while handling codebase", "name", c.Name)
		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	if err := r.updateFinishStatus(ctx, c); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	log.Info("Reconciling codebase has been finished")
	return reconcile.Result{}, nil
}

func (r ReconcileCodebase) updateFinishStatus(ctx context.Context, c *codebaseApi.Codebase) error {
	c.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          codebaseApi.SetupDeploymentTemplates,
		Result:          codebaseApi.Success,
		Value:           "active",
		FailureCount:    0,
		Git:             c.Status.Git,
	}
	return r.updateStatus(ctx, c)
}

// setFailureCount increments failure count and returns delay for next reconciliation
func (r ReconcileCodebase) setFailureCount(c *codebaseApi.Codebase) time.Duration {
	timeout := util.GetTimeout(c.Status.FailureCount, 10*time.Second)
	r.log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	c.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileCodebase) getChain(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
	if r.chainGetter == nil {
		r.chainGetter = func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
			r.log.Info("select correct chain to handle codebase", "name", cr.Name)
			return r.getStrategyChain(cr)
		}
	}

	return r.chainGetter(cr)
}

func (r ReconcileCodebase) getStrategyChain(c *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
	repo := r.createCodebaseRepo(c)
	if c.Spec.Strategy == util.ImportStrategy {
		return r.getCiChain(c, repo)
	}
	return chain.CreateGerritDefChain(r.client, repo), nil
}

func (r ReconcileCodebase) createCodebaseRepo(c *codebaseApi.Codebase) repository.CodebaseRepository {
	if r.db == nil {
		return repository.NewK8SCodebaseRepository(r.client, c)
	}
	return repository.SqlCodebaseRepository{DB: r.db}
}

func (r ReconcileCodebase) getCiChain(c *codebaseApi.Codebase, repo repository.CodebaseRepository) (cHand.CodebaseHandler, error) {
	if strings.ToLower(c.Spec.CiTool) == util.GitlabCi {
		return chain.CreateGitlabCiDefChain(r.client, repo), nil
	}
	return chain.CreateThirdPartyVcsProviderDefChain(r.client, repo), nil
}

func (r *ReconcileCodebase) updateStatus(ctx context.Context, instance *codebaseApi.Codebase) error {
	if err := r.client.Status().Update(ctx, instance); err != nil {
		if err := r.client.Update(ctx, instance); err != nil {
			return err
		}
	}
	return nil
}

func (r ReconcileCodebase) tryToDeleteCodebase(ctx context.Context, c *codebaseApi.Codebase) (*reconcile.Result, error) {
	if c.GetDeletionTimestamp().IsZero() {
		return nil, nil
	}

	if err := removeDirectoryIfExists(c.Name, c.Namespace); err != nil {
		return nil, err
	}

	if err := chain.CreateDeletionChain(r.client).ServeRequest(c); err != nil {
		return nil, errors.Wrap(err, "errors during deletion chain")
	}

	c.ObjectMeta.Finalizers = util.RemoveString(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName)
	if err := r.client.Update(ctx, c); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func removeDirectoryIfExists(codebaseName, namespace string) error {
	wd := util.GetWorkDir(codebaseName, namespace)
	if err := util.RemoveDirectory(wd); err != nil {
		return errors.Wrap(err, "unable to remove directory if exists")
	}
	return nil
}

func (r ReconcileCodebase) setFinalizers(ctx context.Context, c *codebaseApi.Codebase) error {
	if !c.GetDeletionTimestamp().IsZero() {
		return nil
	}

	if !util.ContainsString(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName) {
		c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName)
	}

	if !util.ContainsString(c.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName) {
		c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, util.ForegroundDeletionFinalizerName)
	}

	return r.client.Update(ctx, c)
}
