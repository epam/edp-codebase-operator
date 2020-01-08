package gitserver

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
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

var log = logf.Log.WithName("controller_git_server")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new GitServer Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGitServer{
		Client:     mgr.GetClient(),
		CoreClient: openshift.CreateOpenshiftClients().CoreClient,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gitserver-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*v1alpha1.GitServer)
			newObject := e.ObjectNew.(*v1alpha1.GitServer)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	// Watch for changes to primary resource GitServer
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.GitServer{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileGitServer{}

// ReconcileGitServer reconciles a codebase object
type ReconcileGitServer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client     client.Client
	CoreClient *coreV1Client.CoreV1Client
}

// Reconcile reads that state of the cluster for a GitServer object and makes changes based on the state read
// and what is in the GitServer.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGitServer) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GitServer")

	// Fetch the GitServer instance
	instance := &edpv1alpha1.GitServer{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
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

	gitServer, _ := model.ConvertToGitServer(*instance)

	hasConnection, err := checkConnectionToGitServer(*r.CoreClient, *gitServer)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while checking connection to Git Server %v", gitServer.GitHost))
	}

	err = updateStatus(r.Client, instance, hasConnection)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while updating GitServer status %v", gitServer.GitHost))
	}

	log.Info("Reconciling codebase has been finished")

	return reconcile.Result{}, nil
}

func updateStatus(client client.Client, instance *edpv1alpha1.GitServer, hasConnection bool) error {
	instance.Status = generateStatus(hasConnection)

	err := client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = client.Update(context.TODO(), instance)
	}

	log.Info("Status for GitServer is set up.")

	return nil
}

func generateStatus(hasConnection bool) v1alpha1.GitServerStatus {
	if hasConnection {
		return v1alpha1.GitServerStatus{
			Status:          "created",
			Available:       hasConnection,
			LastTimeUpdated: time.Now(),
			Result:          "success",
			Username:        "system",
			Value:           "active",
		}
	}

	return v1alpha1.GitServerStatus{
		Status:          "created",
		Available:       hasConnection,
		LastTimeUpdated: time.Now(),
		Result:          "error",
		Username:        "system",
		Value:           "inactive",
	}
}
