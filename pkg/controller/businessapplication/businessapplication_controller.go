package businessapplication

import (
	"context"
	"errors"
	"log"
	"strings"

	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"business-app-handler-controller/pkg/controller/businessapplication/impl"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

var allowedAppSettings = map[string][]string{
"add_repo_strategy": {"create", "clone"},
"language":   {"java", "dotnet", "javascript"},
"build_tool": {"maven", "gradle", "dotnet", "npm"},
"framework": {"springboot", "springboot(multi-module)", "netcore", "react"},
}

func containSettings(slice []string, value string) bool {
	for _, element := range slice {
		if element == strings.ToLower(value) {
			return true
		}
	}
	return false
}

 type BusinessApplication interface {
 	Create()
 	Update()
 	Delete()
 }

 func getBusinessApplication(cr *edpv1alpha1.BusinessApplication, r *ReconcileBusinessApplication) (BusinessApplication, error) {
 	if !(containSettings(allowedAppSettings["add_repo_strategy"], string(cr.Spec.Strategy))) {
 		return nil, errors.New("Provided unsupported add repository strategy - " + string(cr.Spec.Strategy))
	} else if !(containSettings(allowedAppSettings["language"], cr.Spec.Lang)) {
		return nil, errors.New("Provided unsupported language - " + cr.Spec.Lang)
	} else if !(containSettings(allowedAppSettings["build_tool"], cr.Spec.BuildTool)) {
		return nil, errors.New("Provided unsupported build tool - " + cr.Spec.BuildTool)
	} else if !(containSettings(allowedAppSettings["framework"], cr.Spec.Framework)) {
		return nil, errors.New("Provided unsupported framework - " + cr.Spec.Framework)
	} else {
 		return impl.BusinessApplication{
 			cr,
 			r.client,
 			r.scheme,
		}, nil
	 }
 }

// Add creates a new BusinessApplication Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBusinessApplication{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("businessapplication-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BusinessApplication
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.BusinessApplication{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileBusinessApplication{}

// ReconcileBusinessApplication reconciles a BusinessApplication object
type ReconcileBusinessApplication struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a BusinessApplication object and makes changes based on the state read
// and what is in the BusinessApplication.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBusinessApplication) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling BusinessApplication %s/%s", request.Namespace, request.Name)

	// Fetch the BusinessApplication instance
	instance := &edpv1alpha1.BusinessApplication{}
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

	businessApplication, err := getBusinessApplication(instance, r)
	if err != nil {
		log.Fatalf("[ERROR] Cannot get Business Application %s. Reason: %s", request.Name, err)
	}
	businessApplication.Create()

	log.Printf("Reconciling BusinessApplication %s/%s has been finished", request.Namespace, request.Name)

	return reconcile.Result{}, nil
}
