package codebase

import (
	"context"
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

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain"
	cHand "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	validate "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/validation"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const codebaseOperatorFinalizerName = "codebase.operator.finalizer.name"

func NewReconcileCodebase(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCodebase {
	return &ReconcileCodebase{
		client: c,
		scheme: scheme,
		log:    log.WithName("codebase"),
	}
}

type ReconcileCodebase struct {
	client      client.Client
	scheme      *runtime.Scheme
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

	err = r.initLabels(ctx, c)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "fail to init labels for codebase")
	}

	ch, err := r.getChain(c)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := ch.ServeRequest(ctx, c); err != nil {
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

func (r *ReconcileCodebase) updateFinishStatus(ctx context.Context, c *codebaseApi.Codebase) error {
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

// setFailureCount increments failure count and returns delay for next reconciliation.
func (r *ReconcileCodebase) setFailureCount(c *codebaseApi.Codebase) time.Duration {
	timeout := util.GetTimeout(c.Status.FailureCount, 10*time.Second)

	r.log.V(2).Info("wait for next reconciliation", "next reconciliation in", timeout)

	c.Status.FailureCount++

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

func (r *ReconcileCodebase) getStrategyChain(c *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
	repo := r.createCodebaseRepo(c)
	if c.Spec.Strategy == util.ImportStrategy {
		return r.getCiChain(c, repo)
	}
	if strings.EqualFold(c.Spec.CiTool, util.Tekton) {
		return chain.MakeGerritTektonChain(r.client, repo), nil
	}
	return chain.MakeGerritDefChain(r.client, repo), nil
}

func (r *ReconcileCodebase) createCodebaseRepo(c *codebaseApi.Codebase) repository.CodebaseRepository {
	return repository.NewK8SCodebaseRepository(r.client, c)
}

func (r *ReconcileCodebase) getCiChain(c *codebaseApi.Codebase, repo repository.CodebaseRepository) (cHand.CodebaseHandler, error) {
	if strings.EqualFold(c.Spec.CiTool, util.GitlabCi) {
		return chain.MakeGitlabCiDefChain(r.client, repo), nil
	}
	if strings.EqualFold(c.Spec.CiTool, util.Tekton) {
		return chain.MakeTektonCiDefChain(r.client, repo), nil
	}
	return chain.MakeThirdPartyVcsProviderDefChain(r.client, repo), nil
}

func (r *ReconcileCodebase) updateStatus(ctx context.Context, instance *codebaseApi.Codebase) error {
	if err := r.client.Status().Update(ctx, instance); err != nil {
		if err := r.client.Update(ctx, instance); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCodebase) tryToDeleteCodebase(ctx context.Context, c *codebaseApi.Codebase) (*reconcile.Result, error) {
	if c.GetDeletionTimestamp().IsZero() {
		return nil, nil
	}

	if err := removeDirectoryIfExists(c.Name, c.Namespace); err != nil {
		return nil, err
	}

	if c.Spec.CiTool != util.Tekton {
		if err := chain.MakeDeletionChain(r.client).ServeRequest(ctx, c); err != nil {
			return nil, errors.Wrap(err, "errors during deletion chain")
		}
	}

	c.ObjectMeta.Finalizers = util.RemoveString(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName)
	if err := r.client.Update(ctx, c); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r *ReconcileCodebase) setFinalizers(ctx context.Context, c *codebaseApi.Codebase) error {
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

func removeDirectoryIfExists(codebaseName, namespace string) error {
	wd := util.GetWorkDir(codebaseName, namespace)
	if err := util.RemoveDirectory(wd); err != nil {
		return errors.Wrap(err, "unable to remove directory if exists")
	}
	return nil
}

func (r *ReconcileCodebase) initLabels(ctx context.Context, c *codebaseApi.Codebase) error {
	const codebaseTypeLabelName = "app.edp.epam.com/codebaseType"

	r.log.Info("Trying to update labels for codebase", "name", c.Name)

	originalCodeBase := c.DeepCopy()

	labels := c.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels[codebaseTypeLabelName]; ok {
		r.log.Info("Codebase already has label", "name", c.Name, "label", codebaseTypeLabelName)
		return nil
	}

	labels[codebaseTypeLabelName] = c.Spec.Type

	c.SetLabels(labels)

	return r.client.Patch(ctx, c, client.MergeFrom(originalCodeBase))
}
