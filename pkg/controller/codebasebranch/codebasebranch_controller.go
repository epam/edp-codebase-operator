package codebasebranch

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/impl"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

var log = logf.Log.WithName("codebase-branch-controller")

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
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CodebaseBranch")

	// Fetch the CodebaseBranch instance
	instance := &edpv1alpha1.CodebaseBranch{}
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

	app, err := r.getApplicationByBranch(*instance)

	if err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second},
			errors.Wrapf(err, "couldn't get codebase for branch %s", request.Name)
	}

	jfn := fmt.Sprintf("%v-%v", app.Name, "jenkins-folder")
	jf, err := r.getJenkinsFolder(jfn, app.Namespace)
	if err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	a := jf == nil || !jf.Status.Available
	if !app.Status.Available && a {
		log.Info("can't start reconcile for branch", "codebase", app.Name,
			"codebase status", app.Status.Available, "branch", request.Name, "jenkins folder", a)
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	codebaseBranch, err := getCodebaseBranchService(r)
	if err != nil {
		log.Error(err, "couldn't initializate codebase branch service")
	}
	codebaseBranch.Create(instance)

	reqLogger.Info("Reconciling CodebaseBranch has been finished")
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

func (r *ReconcileCodebaseBranch) updateStatus(instance *edpv1alpha1.CodebaseBranch) {
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r *ReconcileCodebaseBranch) getJenkinsFolder(name, namespace string) (*v1alpha1.JenkinsFolder, error) {
	i := &v1alpha1.JenkinsFolder{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, i)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get jenkins folder %v", name)
	}
	return i, nil
}
