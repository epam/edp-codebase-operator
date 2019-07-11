package codebasebranch

import (
	edpv1alpha1 "codebase-operator/pkg/apis/edp/v1alpha1"
	"codebase-operator/pkg/controller/codebasebranch/impl"
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

type CodebaseBranchService interface {
	Create(cr *edpv1alpha1.CodebaseBranch)
	Update(cr *edpv1alpha1.CodebaseBranch)
	Delete(cr *edpv1alpha1.CodebaseBranch)
}

// Add creates a new CodebaseBranch Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCodebaseBranch{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("codebasebranch-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CodebaseBranch
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.CodebaseBranch{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCodebaseBranch{}

// ReconcileCodebaseBranch reconciles a CodebaseBranch object
type ReconcileCodebaseBranch struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CodebaseBranch object and makes changes based on the state read
// and what is in the CodebaseBranch.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCodebaseBranch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling CodebaseBranch %s/%s", request.Namespace, request.Name)

	// Fetch the CodebaseBranch instance
	instance := &edpv1alpha1.CodebaseBranch{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	app, err := r.getApplicationByBranch(*instance)

	if err != nil {
		log.Printf("[ERROR] Cannot get codebase for branch %s. Reason: %s", request.Name, err)
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	if !app.Status.Available {
		log.Printf("[ERROR] Codebase %s for branch %s is not ready yet.", app.Name, request.Name)
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	codebaseBranch, err := getCodebaseBranchService(r)
	if err != nil {
		log.Fatalf("[ERROR] Cannot get codebase branch %s. Reason: %s", request.Name, err)
	}
	codebaseBranch.Create(instance)
	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		log.Printf("[ERROR] Cannot update status of the branch: %v status", instance.Name)
	}

	log.Printf("Reconciling CodebaseBranch %s/%s has been finished", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) getApplicationByBranch(branch edpv1alpha1.CodebaseBranch) (*edpv1alpha1.Codebase, error) {
	instance := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: branch.Namespace,
		Name:      branch.Spec.CodebaseName,
	}, instance)

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func getCodebaseBranchService(r *ReconcileCodebaseBranch) (CodebaseBranchService, error) {
	return impl.CodebaseBranchService{
		r.client,
	}, nil
}
