package codebase

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/db"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain"
	cHand "github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	validate "github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/validation"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	perfApi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

var log = logf.Log.WithName("controller_codebase")

type CodebaseService interface {
	Create() error
	Update()
	Delete()
}

// Add creates a new Codebase Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileCodebase{
		client: mgr.GetClient(),
		scheme: scheme,
		db:     db.GetConnection(),
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	schemeGroupVersionV1 := schema.GroupVersion{Group: "v1.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersionV1,
		&v1alpha1.EDPComponent{},
		&v1alpha1.EDPComponentList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersionV1)

	schemeGroupVersionV2 := schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersionV2,
		&perfApi.PerfDataSourceJenkins{},
		&perfApi.PerfDataSourceJenkinsList{},
		&perfApi.PerfDataSourceSonar{},
		&perfApi.PerfDataSourceSonarList{},
		&perfApi.PerfDataSourceGitLab{},
		&perfApi.PerfDataSourceGitLabList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersionV2)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("codebase-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.Codebase)
			no := e.ObjectNew.(*edpv1alpha1.Codebase)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource Codebase
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Codebase{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCodebase{}

const (
	foregroundDeletionFinalizerName = "foregroundDeletion"
	codebaseOperatorFinalizerName   = "codebase.operator.finalizer.name"
)

// ReconcileCodebase reconciles a codebase object
type ReconcileCodebase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	handler cHand.CodebaseHandler
	db      *sql.DB
}

// Reconcile reads that state of the cluster for a Codebase object and makes changes based on the state read
// and what is in the Codebase.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCodebase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Codebase")

	// Fetch the Codebase instance
	c := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), request.NamespacedName, c)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	defer r.updateStatus(c)

	result, err := r.tryToDeleteCodebase(c)
	if err != nil || result != nil {
		return *result, err
	}

	if !validate.IsCodebaseValid(c) {
		return reconcile.Result{}, nil
	}

	ch, err := r.getChain(c)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := ch.ServeRequest(c); err != nil {
		timeout := setFailureCount(c)
		log.Error(err, "an error has occurred while handling codebase", "name", c.Name)
		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	if err := r.updateFinishStatus(c); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	reqLogger.Info("Reconciling codebase has been finished")
	return reconcile.Result{}, nil
}

func (r ReconcileCodebase) updateFinishStatus(c *edpv1alpha1.Codebase) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
		FailureCount:    0,
		Git:             c.Status.Git,
	}
	return r.updateStatus(c)
}

// setFailureCount increments failure count and returns delay for next reconciliation
func setFailureCount(c *edpv1alpha1.Codebase) time.Duration {
	timeout := util.GetTimeout(c.Status.FailureCount)
	log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	c.Status.FailureCount += 1
	return timeout
}

func (r ReconcileCodebase) getChain(cr *edpv1alpha1.Codebase) (cHand.CodebaseHandler, error) {
	log.Info("select correct chain to handle codebase", "name", cr.Name)
	return r.getStrategyChain(cr)
}

func (r ReconcileCodebase) getStrategyChain(c *edpv1alpha1.Codebase) (cHand.CodebaseHandler, error) {
	cs := openshift.CreateOpenshiftClients()
	cs.Client = r.client
	repo := r.createCodebaseRepo(c)
	if c.Spec.Strategy == util.ImportStrategy {
		return r.getCiChain(c, cs, repo)
	}
	return chain.CreateGerritDefChain(*cs, repo), nil
}

func (r ReconcileCodebase) createCodebaseRepo(c *edpv1alpha1.Codebase) repository.CodebaseRepository {
	if r.db == nil {
		return repository.NewK8SCodebaseRepository(r.client, c)
	}
	return repository.SqlCodebaseRepository{DB: r.db}
}

func (r ReconcileCodebase) getCiChain(c *edpv1alpha1.Codebase, cs *openshift.ClientSet,
	repo repository.CodebaseRepository) (cHand.CodebaseHandler, error) {
	if strings.ToLower(c.Spec.CiTool) == util.GitlabCi {
		return chain.CreateGitlabCiDefChain(*cs, repo), nil
	}
	return chain.CreateThirdPartyVcsProviderDefChain(*cs, repo), nil
}

func (r *ReconcileCodebase) updateStatus(instance *edpv1alpha1.Codebase) error {
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		if err := r.client.Update(context.TODO(), instance); err != nil {
			return err
		}
	}
	return nil
}

func (r ReconcileCodebase) tryToDeleteCodebase(c *edpv1alpha1.Codebase) (*reconcile.Result, error) {
	if c.GetDeletionTimestamp().IsZero() {
		if !util.ContainsString(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName) {
			c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName)
			if err := r.client.Update(context.TODO(), c); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return nil, nil
	}

	if err := removeDirectoryIfExists(c.Name, c.Namespace); err != nil {
		return &reconcile.Result{}, err
	}

	c.ObjectMeta.Finalizers = util.RemoveString(c.ObjectMeta.Finalizers, codebaseOperatorFinalizerName)
	if err := r.client.Update(context.TODO(), c); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func removeDirectoryIfExists(codebaseName, namespace string) error {
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", namespace, codebaseName)
	if !util.DoesDirectoryExist(wd) {
		return nil
	}
	if err := util.RemoveDirectory(wd); err != nil {
		return err
	}
	return nil
}
