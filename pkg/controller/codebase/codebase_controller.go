package codebase

import (
	"context"
	"errors"
	"log"
	"strings"

	edpv1alpha1 "codebase-operator/pkg/apis/edp/v1alpha1"
	"codebase-operator/pkg/controller/codebase/impl"
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

var allowedCodebaseSettings = map[string][]string{
	"add_repo_strategy": {"create", "clone"},
	"language":          {"java", "dotnet", "javascript"},
	"build_tool":        {"maven", "gradle", "dotnet", "npm"},
	"framework":         {"springboot", "springboot(multi-module)", "netcore", "react"},
}

func containSettings(slice []string, value string) bool {
	for _, element := range slice {
		if element == strings.ToLower(value) {
			return true
		}
	}
	return false
}

type CodebaseService interface {
	Create()
	Update()
	Delete()
}

func getCodebase(cr *edpv1alpha1.Codebase, r *ReconcileCodebase) (CodebaseService, error) {
	if !(containSettings(allowedCodebaseSettings["add_repo_strategy"], string(cr.Spec.Strategy))) {
		return nil, errors.New("Provided unsupported add repository strategy - " + string(cr.Spec.Strategy))
	} else if !(containSettings(allowedCodebaseSettings["language"], cr.Spec.Lang)) {
		return nil, errors.New("Provided unsupported language - " + cr.Spec.Lang)
	} else if !(containSettings(allowedCodebaseSettings["build_tool"], cr.Spec.BuildTool)) {
		return nil, errors.New("Provided unsupported build tool - " + cr.Spec.BuildTool)
	} else {
		return impl.CodebaseService{
			cr,
			r.client,
			r.scheme,
		}, nil
	}
}

// Add creates a new Codebase Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCodebase{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("codebase-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Codebase
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Codebase{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCodebase{}

// ReconcileCodebase reconciles a codebase object
type ReconcileCodebase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Codebase object and makes changes based on the state read
// and what is in the Codebase.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCodebase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Codebase %s/%s", request.Namespace, request.Name)

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

	codebase, err := getCodebase(instance, r)
	if err != nil {
		log.Fatalf("[ERROR] Cannot get codebase %s. Reason: %s", request.Name, err)
	}
	codebase.Create()
	_ = r.client.Update(context.TODO(), instance)

	log.Printf("Reconciling codebase %s/%s has been finished", request.Namespace, request.Name)

	return reconcile.Result{}, nil
}
