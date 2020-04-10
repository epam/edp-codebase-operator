package codebase

import (
	"context"
	"database/sql"
	"github.com/epmd-edp/codebase-operator/v2/db"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain"
	cHand "github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	validate "github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/validation"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
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
	return &ReconcileCodebase{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		db:     db.GetConnection(),
	}
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

const ForegroundDeletionFinalizerName = "foregroundDeletion"

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
	instance := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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
	defer r.updateStatus(instance)

	if err := r.tryToAddFinalizer(instance); err != nil {
		return reconcile.Result{}, err
	}

	if !validate.IsCodebaseValid(instance) {
		return reconcile.Result{}, nil
	}

	ch, err := r.getChain(instance)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := ch.ServeRequest(instance); err != nil {
		timeout := setFailureCount(instance)
		log.Error(err, "an error has occurred while handling codebase", "name", instance.Name)
		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	reqLogger.Info("Reconciling codebase has been finished")
	return reconcile.Result{}, nil
}

// setFailureCount increments failure count and returns delay for next reconciliation
func setFailureCount(c *edpv1alpha1.Codebase) time.Duration {
	timeout := getTimeout(c.Status.FailureCount)
	log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	c.Status.FailureCount += 1
	return timeout
}

func getTimeout(factor int64) time.Duration {
	t := float64(500*time.Millisecond) * math.Pow(math.E, float64(factor+1))
	return time.Duration(t)
}

func (r ReconcileCodebase) getChain(cr *edpv1alpha1.Codebase) (cHand.CodebaseHandler, error) {
	log.Info("select correct chain to handle codebase", "name", cr.Name)
	cs := openshift.CreateOpenshiftClients()
	cs.Client = r.client
	if cr.Spec.Strategy == util.ImportStrategy {
		return chain.CreateThirdPartyVcsProviderDefChain(*cs, r.db), nil
	}
	return chain.CreateGerritDefChain(*cs, r.db), nil
}

func (r *ReconcileCodebase) updateStatus(instance *edpv1alpha1.Codebase) {
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r ReconcileCodebase) tryToAddFinalizer(c *edpv1alpha1.Codebase) error {
	if !util.ContainsString(c.ObjectMeta.Finalizers, ForegroundDeletionFinalizerName) {
		c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, ForegroundDeletionFinalizerName)
		if err := r.client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
