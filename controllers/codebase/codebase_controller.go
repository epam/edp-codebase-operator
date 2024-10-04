package codebase

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain"
	cHand "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/objectmodifier"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const codebaseOperatorFinalizerName = "codebase.operator.finalizer.name"

func NewReconcileCodebase(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCodebase {
	return &ReconcileCodebase{
		client:   c,
		scheme:   scheme,
		log:      log.WithName("codebase"),
		modifier: objectmodifier.NewCodebaseModifier(c),
	}
}

type ReconcileCodebase struct {
	client      client.Client
	scheme      *runtime.Scheme
	log         logr.Logger
	chainGetter func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error)
	modifier    *objectmodifier.CodebaseModifier
}

func (r *ReconcileCodebase) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*codebaseApi.Codebase)
			if !ok {
				return false
			}

			no, ok := e.ObjectNew.(*codebaseApi.Codebase)
			if !ok {
				return false
			}

			if codebasepredicate.PauseAnnotationChanged(oo, no) {
				return true
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

	pause := codebasepredicate.NewPause(r.log)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.Codebase{}, builder.WithPredicates(pause, p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build Codebase controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebases/finalizers,verbs=update
//+kubebuilder:rbac:groups=route.openshift.io,namespace=placeholder,resources=routes,verbs=get;list;watch

// Reconcile reads that state of the cluster for a Codebase object and makes changes based on the state.
func (r *ReconcileCodebase) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling Codebase")

	codebase := &codebaseApi.Codebase{}
	if err := r.client.Get(ctx, request.NamespacedName, codebase); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch Codebase resource %q: %w", request.NamespacedName, err)
	}

	patched, err := r.modifier.Apply(ctx, codebase)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to apply codebase changes: %w", err)
	}

	if patched {
		log.Info("codebase default values has been patched")

		return reconcile.Result{}, nil
	}

	if err = r.setFinalizers(ctx, codebase); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to set finalizers: %w", err)
	}

	result, err := r.tryToDeleteCodebase(ctx, codebase)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to try to delete codebase: %w", err)
	}

	if result != nil {
		return *result, nil
	}

	err = r.initLabels(ctx, codebase)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to init labels for codebase: %w", err)
	}

	ch, err := r.getChain(ctx, codebase)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to select chain: %w", err)
	}

	if err = ch.ServeRequest(ctx, codebase); err != nil {
		timeout := r.setFailureCount(ctx, codebase)

		log.Error(err, "Error during codebase reconciliation")
		log.Info("Wait for next reconciliation", "timeout", timeout)

		if cleanErr := removeDirectoryIfExists(ctx, codebase); cleanErr != nil {
			log.Error(cleanErr, "Failed to remove Codebase directory")
		}

		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	if err := r.updateFinishStatus(ctx, codebase); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update %v Codebase status: %w", codebase.Name, err)
	}

	log.Info("Reconciling Codebase has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileCodebase) updateFinishStatus(ctx context.Context, c *codebaseApi.Codebase) error {
	// Set WebHookRef from WebHookID for backward compatibility.
	webHookRef := c.Status.WebHookRef
	if webHookRef == "" && c.Status.WebHookID != 0 {
		webHookRef = strconv.Itoa(c.Status.WebHookID)
	}

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
		WebHookID:       c.Status.WebHookID,
		WebHookRef:      webHookRef,
		GitWebUrl:       c.Status.GitWebUrl,
	}

	if err := r.client.Status().Update(ctx, c); err != nil {
		return fmt.Errorf("failed to update Codebase status: %w", err)
	}

	return nil
}

// setFailureCount increments failure count and returns delay for next reconciliation.
func (r *ReconcileCodebase) setFailureCount(ctx context.Context, codebase *codebaseApi.Codebase) time.Duration {
	const defaultTimeout = 10 * time.Second
	timeout := util.GetTimeout(codebase.Status.FailureCount, defaultTimeout)

	codebase.Status.FailureCount++

	if err := r.client.Status().Update(ctx, codebase); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to update Codebase status with failure count")
	}

	return timeout
}

func (r *ReconcileCodebase) getChain(ctx context.Context, codebase *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
	if r.chainGetter == nil {
		r.chainGetter = func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
			return chain.MakeChain(ctx, r.client), nil
		}
	}

	return r.chainGetter(codebase)
}

func (r *ReconcileCodebase) tryToDeleteCodebase(ctx context.Context, codebase *codebaseApi.Codebase) (*reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if codebase.GetDeletionTimestamp().IsZero() {
		return nil, nil
	}

	log.Info("Deleting codebase")

	if err := removeDirectoryIfExists(ctx, codebase); err != nil {
		return nil, err
	}

	if err := chain.MakeDeletionChain(ctx, r.client).ServeRequest(ctx, codebase); err != nil {
		return nil, fmt.Errorf("failed to make deletion chain: %w", err)
	}

	log.Info("Codebase deletion chain has been finished successfully")
	log.Info("Removing finalizer from Codebase", "finalizer", codebaseOperatorFinalizerName)

	controllerutil.RemoveFinalizer(codebase, codebaseOperatorFinalizerName)

	if err := r.client.Update(ctx, codebase); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to update 'Codebase' resource %q: %w", codebase.Name, err)
	}

	log.Info("Codebase has been deleted successfully")

	return &reconcile.Result{}, nil
}

func (r *ReconcileCodebase) setFinalizers(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if !c.GetDeletionTimestamp().IsZero() {
		return nil
	}

	finalizerAdded := false

	if controllerutil.AddFinalizer(c, codebaseOperatorFinalizerName) {
		finalizerAdded = true

		log.Info("Adding finalizer to Codebase", "finalizer", codebaseOperatorFinalizerName)
	}

	if controllerutil.AddFinalizer(c, util.ForegroundDeletionFinalizerName) {
		finalizerAdded = true

		log.Info("Adding finalizer to Codebase", "finalizer", util.ForegroundDeletionFinalizerName)
	}

	if finalizerAdded {
		err := r.client.Update(ctx, c)
		if err != nil {
			return fmt.Errorf("failed to update 'Codebase' resource %q: %w", c.Name, err)
		}

		log.Info("Finalizers were added successfully")
	}

	return nil
}

func removeDirectoryIfExists(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Removing codebase directory before deletion")

	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)
	if err := util.RemoveDirectory(wd); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", wd, err)
	}

	log.Info("Codebase directory was removed successfully", "directory", wd)

	return nil
}

func (r *ReconcileCodebase) initLabels(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Trying to update labels for codebase")

	const codebaseTypeLabelName = "app.edp.epam.com/codebaseType"

	originalCodeBase := c.DeepCopy()

	labels := c.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels[codebaseTypeLabelName]; ok {
		log.Info("Codebase already has label", "label", codebaseTypeLabelName)
		return nil
	}

	labels[codebaseTypeLabelName] = c.Spec.Type

	c.SetLabels(labels)

	err := r.client.Patch(ctx, c, client.MergeFrom(originalCodeBase))
	if err != nil {
		return fmt.Errorf("failed to patch 'Codebase' resource %q: %w", c.Name, err)
	}

	return nil
}
