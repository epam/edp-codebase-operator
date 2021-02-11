package codebasebranch

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/factory"
	cbHandler "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	perfApi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("codebase-branch-controller")

const (
	errorStatus  = "error"
)

type CodebaseBranchService interface {
	Create(cr *edpv1alpha1.CodebaseBranch)
	Update(cr *edpv1alpha1.CodebaseBranch)
	Delete(cr *edpv1alpha1.CodebaseBranch)
}

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileCodebaseBranch{
		client: mgr.GetClient(),
		scheme: scheme,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	schemeGroupVersion := schema.GroupVersion{Group: "v1.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersion,
		&v1alpha1.EDPComponent{},
		&v1alpha1.EDPComponentList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersion)

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

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("codebasebranch-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.CodebaseBranch)
			no := e.ObjectNew.(*edpv1alpha1.CodebaseBranch)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}

	err = c.Watch(&source.Kind{Type: &edpv1alpha1.CodebaseBranch{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCodebaseBranch{}

type ReconcileCodebaseBranch struct {
	client client.Client
	scheme *runtime.Scheme
}

const codebaseBranchOperatorFinalizerName = "codebase.branch.operator.finalizer.name"

func (r *ReconcileCodebaseBranch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rl := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rl.Info("Reconciling CodebaseBranch")

	cb := &edpv1alpha1.CodebaseBranch{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, cb); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer func() {
		if err := r.updateStatus(cb); err != nil {
			rl.Error(err, "error on codebase branch update status")
		}
	}()

	c, err := util.GetCodebase(r.client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.setOwnerRef(cb,c); err != nil {
		setErrorStatus(cb, err.Error())
		return reconcile.Result{}, err
	}

	result, err := r.tryToDeleteCodebaseBranch(cb, factory.GetDeletionChain(c.Spec.CiTool, r.client))
	if err != nil {
		return reconcile.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	cbChain := factory.GetChain(c.Spec.CiTool, r.client)
	if err := cbChain.ServeRequest(cb); err != nil {
		log.Error(err, "an error has occurred while handling codebase branch", "name", cb.Name)
		switch err.(type) {
		case *util.CodebaseBranchReconcileError:
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		default:
			return reconcile.Result{}, err
		}
	}

	if err := r.setSuccessStatus(cb, edpv1alpha1.JenkinsConfiguration); err != nil {
		return reconcile.Result{},
			errors.Wrapf(err, "an error has been occurred while updating %v Codebase branch status", cb.Name)
	}

	rl.Info("Reconciling CodebaseBranch has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) setSuccessStatus(cb *edpv1alpha1.CodebaseBranch, action edpv1alpha1.ActionType) error {
	cb.Status = edpv1alpha1.CodebaseBranchStatus{
		LastTimeUpdated:     time.Now(),
		Username:            "system",
		Action:              cb.Status.Action,
		Result:              edpv1alpha1.Success,
		Value:               "active",
		Status:              model.StatusFinished,
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}
	return r.updateStatus(cb)
}

func (r *ReconcileCodebaseBranch) updateStatus(cb *edpv1alpha1.CodebaseBranch) error {
	if err := r.client.Status().Update(context.TODO(), cb); err != nil {
		if err := r.client.Update(context.TODO(), cb); err != nil {
			return errors.Wrap(err, "couldn't update codebase branch status")
		}
	}
	log.V(2).Info("codebase branch status has been updated", "name", cb.Name)
	return nil
}

func (r ReconcileCodebaseBranch) tryToDeleteCodebaseBranch(cb *edpv1alpha1.CodebaseBranch,
	deletionChain cbHandler.CodebaseBranchHandler) (*reconcile.Result, error) {
	if cb.GetDeletionTimestamp().IsZero() {
		if !util.ContainsString(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName) {
			cb.ObjectMeta.Finalizers = append(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName)
			if err := r.client.Update(context.TODO(), cb); err != nil {
				return &reconcile.Result{}, errors.Wrap(err, "unable to update codebase branch")
			}
		}
		return nil, nil
	}

	if err := deletionChain.ServeRequest(cb); err != nil {
		switch errors.Cause(err).(type) {
		case service.JobFailedError:
			log.Error(err, "deletion job failed")
			return &reconcile.Result{RequeueAfter: setFailureCount(cb)}, nil
		default:
			return nil, errors.Wrap(err, "error during deletion chain")
		}
	}

	if err := removeDirectoryIfExists(cb.Spec.CodebaseName, cb.Name, cb.Namespace); err != nil {
		return &reconcile.Result{}, errors.Wrap(err, "unable to remove codebase branch directory")
	}

	cb.ObjectMeta.Finalizers = util.RemoveString(cb.ObjectMeta.Finalizers, codebaseBranchOperatorFinalizerName)
	if err := r.client.Update(context.TODO(), cb); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func removeDirectoryIfExists(codebaseName, branchName, namespace string) error {
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v", namespace, codebaseName, branchName)
	if !util.DoesDirectoryExist(wd) {
		return nil
	}
	if err := util.RemoveDirectory(wd); err != nil {
		return err
	}
	return nil
}

// setFailureCount increments failure count and returns delay for next reconciliation
func setFailureCount(c *edpv1alpha1.CodebaseBranch) time.Duration {
	timeout := util.GetTimeout(c.Status.FailureCount)
	log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	c.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileCodebaseBranch) setOwnerRef(cb *edpv1alpha1.CodebaseBranch, c *edpv1alpha1.Codebase) error {
	if err := controllerutil.SetControllerReference(c, cb, r.scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for CodebaseBranch CR")
	}
	return nil
}

func setErrorStatus(metadata *edpv1alpha1.CodebaseBranch, msg string) {
	metadata.Status.Status = errorStatus
	metadata.Status.DetailedMessage = msg
}
