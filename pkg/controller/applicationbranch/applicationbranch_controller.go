package applicationbranch

import (
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"business-app-handler-controller/pkg/controller/applicationbranch/impl"
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

type ApplicationBranchService interface {
	Create(cr *edpv1alpha1.ApplicationBranch)
	Update(cr *edpv1alpha1.ApplicationBranch)
	Delete(cr *edpv1alpha1.ApplicationBranch)
}

// Add creates a new ApplicationBranch Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApplicationBranch{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("applicationbranch-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ApplicationBranch
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.ApplicationBranch{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApplicationBranch{}

// ReconcileApplicationBranch reconciles a ApplicationBranch object
type ReconcileApplicationBranch struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ApplicationBranch object and makes changes based on the state read
// and what is in the ApplicationBranch.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileApplicationBranch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling ApplicationBranch %s/%s", request.Namespace, request.Name)

	// Fetch the ApplicationBranch instance
	instance := &edpv1alpha1.ApplicationBranch{}
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

	applicationBranch, err := getApplicationBranchService(r)
	if err != nil {
		log.Fatalf("[ERROR] Cannot get application branch %s. Reason: %s", request.Name, err)
	}
	applicationBranch.Create(instance)
	_ = r.client.Update(context.TODO(), instance)

	log.Printf("Reconciling ApplicationBranch %s/%s has been finished", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}

func getApplicationBranchService(r *ReconcileApplicationBranch) (ApplicationBranchService, error) {
	return impl.ApplicationBranchService{
		r.client,
	}, nil
}
